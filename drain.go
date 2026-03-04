package drain3

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

// Drain is the core log template mining engine.
// It maintains a prefix tree for fast matching and a collection of log clusters.
type Drain struct {
	mu sync.RWMutex

	// Configuration
	SimTh                    float64          `json:"-"`
	Depth                    int              `json:"-"`
	MaxChildren              int              `json:"-"`
	MaxClusters              int              `json:"-"`
	ExtraDelimiters          []string         `json:"-"`
	extraDelimiterRegexps    []*regexp.Regexp `json:"-"`
	ParamStr                 string           `json:"-"`
	ParametrizeNumericTokens bool             `json:"-"`

	// State
	RootNode        *Node        `json:"root_node"`
	IDToCluster     ClusterStore `json:"-"`
	ClustersCounter int          `json:"clusters_counter"`
}

// DrainConfig holds configuration for creating a new Drain instance.
type DrainConfig struct {
	SimTh                    float64
	Depth                    int
	MaxChildren              int
	MaxClusters              int
	ExtraDelimiters          []string
	ParamStr                 string
	ParametrizeNumericTokens bool
}

// DefaultDrainConfig returns a DrainConfig with Python Drain3 defaults.
func DefaultDrainConfig() DrainConfig {
	return DrainConfig{
		SimTh:                    0.4,
		Depth:                    4,
		MaxChildren:              100,
		MaxClusters:              0, // 0 = unlimited
		ExtraDelimiters:          nil,
		ParamStr:                 DefaultParamStr,
		ParametrizeNumericTokens: true,
	}
}

// NewDrain creates a new Drain engine with the given configuration.
func NewDrain(cfg DrainConfig) *Drain {
	if cfg.ParamStr == "" {
		cfg.ParamStr = DefaultParamStr
	}
	if cfg.Depth < 3 {
		cfg.Depth = 4
	}
	if cfg.MaxChildren < 2 {
		cfg.MaxChildren = 100
	}

	var store ClusterStore
	if cfg.MaxClusters > 0 {
		store = NewLogClusterCache(cfg.MaxClusters)
	} else {
		store = newMapStore()
	}

	// Pre-compile extra delimiter regexps (Python uses re.sub for these)
	var delimRegexps []*regexp.Regexp
	for _, delim := range cfg.ExtraDelimiters {
		re, err := regexp.Compile(delim)
		if err != nil {
			// Fallback: treat as literal string by escaping
			re = regexp.MustCompile(regexp.QuoteMeta(delim))
		}
		delimRegexps = append(delimRegexps, re)
	}

	return &Drain{
		SimTh:                    cfg.SimTh,
		Depth:                    cfg.Depth,
		MaxChildren:              cfg.MaxChildren,
		MaxClusters:              cfg.MaxClusters,
		ExtraDelimiters:          cfg.ExtraDelimiters,
		extraDelimiterRegexps:    delimRegexps,
		ParamStr:                 cfg.ParamStr,
		ParametrizeNumericTokens: cfg.ParametrizeNumericTokens,
		RootNode:                 NewNode(),
		IDToCluster:              store,
		ClustersCounter:          0,
	}
}

// maxNodeDepth returns the effective tree depth (Depth - 2, as in Python Drain3).
func (d *Drain) maxNodeDepth() int {
	return d.Depth - 2
}

// AddLogMessage processes a log message and returns the matching cluster and change type.
func (d *Drain) AddLogMessage(content string) (*LogCluster, ChangeType) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.addLogMessageUnlocked(content)
}

// addLogMessageUnlocked is the lock-free internal implementation.
// Callers must hold d.mu (write lock).
func (d *Drain) addLogMessageUnlocked(content string) (*LogCluster, ChangeType) {
	contentTokens := d.GetContentAsTokens(content)

	matchCluster := d.treeSearch(d.RootNode, contentTokens, d.SimTh, false)

	if matchCluster == nil {
		// Create new cluster
		d.ClustersCounter++
		clusterID := d.ClustersCounter
		tokens := make([]string, len(contentTokens))
		copy(tokens, contentTokens)
		matchCluster = NewLogCluster(tokens, clusterID)

		evicted := d.putCluster(matchCluster)
		if evicted != nil {
			d.removeClusterFromTree(evicted)
		}

		d.addSeqToPrefixTree(d.RootNode, matchCluster)
		return matchCluster, ChangeClusterCreated
	}

	// Update existing cluster
	newTemplateTokens := d.createTemplate(contentTokens, matchCluster.LogTemplateTokens)
	matchCluster.Size++

	if !tokensEqual(matchCluster.LogTemplateTokens, newTemplateTokens) {
		matchCluster.LogTemplateTokens = newTemplateTokens
		// Touch the cluster in LRU to update its position
		if lru, ok := d.IDToCluster.(*LogClusterCache); ok {
			lru.Touch(matchCluster.ClusterID)
		}
		return matchCluster, ChangeClusterTemplateChanged
	}

	// Touch the cluster in LRU
	if lru, ok := d.IDToCluster.(*LogClusterCache); ok {
		lru.Touch(matchCluster.ClusterID)
	}
	return matchCluster, ChangeNone
}

// Match finds the best matching cluster for a log message without modifying any state.
func (d *Drain) Match(content string, fullSearchStrategy SearchStrategy) *LogCluster {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.matchUnlocked(content, fullSearchStrategy)
}

// matchUnlocked is the lock-free internal implementation.
// Callers must hold d.mu (at least read lock).
func (d *Drain) matchUnlocked(content string, fullSearchStrategy SearchStrategy) *LogCluster {
	requiredSimTh := d.SimTh
	contentTokens := d.GetContentAsTokens(content)

	fullSearch := func() *LogCluster {
		allIDs := d.getClustersIDsForSeqLen(len(contentTokens))
		return d.fastMatch(allIDs, contentTokens, requiredSimTh, true)
	}

	if fullSearchStrategy == SearchStrategyAlways {
		return fullSearch()
	}

	matchCluster := d.treeSearch(d.RootNode, contentTokens, requiredSimTh, true)
	if matchCluster != nil {
		return matchCluster
	}

	if fullSearchStrategy == SearchStrategyNever {
		return nil
	}

	return fullSearch()
}

// GetContentAsTokens splits content into tokens, applying extra delimiters.
// Extra delimiters are treated as regex patterns (matching Python's re.sub behavior).
func (d *Drain) GetContentAsTokens(content string) []string {
	for _, re := range d.extraDelimiterRegexps {
		content = re.ReplaceAllString(content, " ")
	}
	return strings.Fields(content)
}

// treeSearch navigates the prefix tree to find the best matching cluster.
func (d *Drain) treeSearch(rootNode *Node, tokens []string, simTh float64, includeParams bool) *LogCluster {
	tokenCount := len(tokens)

	// First level: children are keyed by token count
	curNode := rootNode.KeyToChildNode[strconv.Itoa(tokenCount)]
	if curNode == nil {
		return nil
	}

	// Handle empty log string
	if tokenCount == 0 {
		if len(curNode.ClusterIDs) > 0 {
			return d.IDToCluster.Get(curNode.ClusterIDs[0])
		}
		return nil
	}

	// Navigate tree by token prefix, up to maxNodeDepth
	curNodeDepth := 1
	for _, token := range tokens {
		if curNodeDepth >= d.maxNodeDepth() {
			break
		}
		if curNodeDepth >= tokenCount {
			break
		}

		keyToChildNode := curNode.KeyToChildNode
		nextNode, ok := keyToChildNode[token]
		if ok {
			curNode = nextNode
		} else {
			nextNode, ok = keyToChildNode[d.ParamStr]
			if ok {
				curNode = nextNode
			} else {
				return nil
			}
		}
		curNodeDepth++
	}

	// Find best match among candidates at this leaf node
	return d.fastMatch(curNode.ClusterIDs, tokens, simTh, includeParams)
}

// fastMatch finds the best matching cluster from a list of candidates.
func (d *Drain) fastMatch(clusterIDs []int, tokens []string, simTh float64, includeParams bool) *LogCluster {
	var bestMatch *LogCluster
	bestSim := float64(-1)
	bestParamCount := 0

	for _, clusterID := range clusterIDs {
		cluster := d.IDToCluster.Get(clusterID)
		if cluster == nil {
			continue
		}
		sim, paramCount := d.getSeqDistance(cluster.LogTemplateTokens, tokens, includeParams)
		if sim > bestSim || (sim == bestSim && paramCount < bestParamCount) {
			bestSim = sim
			bestParamCount = paramCount
			bestMatch = cluster
		}
	}

	if bestSim >= simTh {
		return bestMatch
	}
	return nil
}

// getSeqDistance computes the similarity between a template and a log message.
// Returns (similarity, paramCount).
func (d *Drain) getSeqDistance(seq1, seq2 []string, includeParams bool) (float64, int) {
	if len(seq1) != len(seq2) {
		return 0, 0
	}

	// Empty sequences are a perfect match
	if len(seq1) == 0 {
		return 1.0, 0
	}

	simTokens := 0
	paramCount := 0
	for i, token1 := range seq1 {
		token2 := seq2[i]
		if token1 == d.ParamStr {
			paramCount++
			continue
		}
		if token1 == token2 {
			simTokens++
		}
	}

	retVal := float64(simTokens) / float64(len(seq1))

	if includeParams {
		retVal = float64(simTokens+paramCount) / float64(len(seq1))
	}

	return retVal, paramCount
}

// createTemplate merges a log message with an existing template.
// Where tokens differ, the token is replaced with ParamStr.
func (d *Drain) createTemplate(seq1, seq2 []string) []string {
	if len(seq1) != len(seq2) {
		// Should not happen in practice since we match by token count
		return seq2
	}

	retVal := make([]string, len(seq2))
	for i := range seq2 {
		if seq1[i] == seq2[i] {
			retVal[i] = seq2[i]
		} else {
			retVal[i] = d.ParamStr
		}
	}
	return retVal
}

// addSeqToPrefixTree inserts a cluster into the prefix tree.
func (d *Drain) addSeqToPrefixTree(rootNode *Node, cluster *LogCluster) {
	tokenCount := len(cluster.LogTemplateTokens)
	tokenCountStr := strconv.Itoa(tokenCount)

	firstLayerNode, ok := rootNode.KeyToChildNode[tokenCountStr]
	if !ok {
		firstLayerNode = NewNode()
		rootNode.KeyToChildNode[tokenCountStr] = firstLayerNode
	}

	curNode := firstLayerNode

	// Handle empty log string
	if tokenCount == 0 {
		curNode.ClusterIDs = []int{cluster.ClusterID}
		return
	}

	currentDepth := 1
	for _, token := range cluster.LogTemplateTokens {
		// If at max depth or last token — add cluster to leaf
		if currentDepth >= d.maxNodeDepth() || currentDepth >= tokenCount {
			// Clean up stale cluster IDs
			newClusterIDs := make([]int, 0, len(curNode.ClusterIDs)+1)
			for _, cid := range curNode.ClusterIDs {
				if d.IDToCluster.Get(cid) != nil {
					newClusterIDs = append(newClusterIDs, cid)
				}
			}
			newClusterIDs = append(newClusterIDs, cluster.ClusterID)
			curNode.ClusterIDs = newClusterIDs
			break
		}

		// Navigate or create child node
		if _, exists := curNode.KeyToChildNode[token]; !exists {
			if d.ParametrizeNumericTokens && hasNumbers(token) {
				// Numeric token: route to param node
				if _, paramExists := curNode.KeyToChildNode[d.ParamStr]; !paramExists {
					newNode := NewNode()
					curNode.KeyToChildNode[d.ParamStr] = newNode
					curNode = newNode
				} else {
					curNode = curNode.KeyToChildNode[d.ParamStr]
				}
			} else {
				// Non-numeric token
				if _, paramExists := curNode.KeyToChildNode[d.ParamStr]; paramExists {
					if len(curNode.KeyToChildNode) < d.MaxChildren {
						newNode := NewNode()
						curNode.KeyToChildNode[token] = newNode
						curNode = newNode
					} else {
						curNode = curNode.KeyToChildNode[d.ParamStr]
					}
				} else {
					if len(curNode.KeyToChildNode)+1 < d.MaxChildren {
						newNode := NewNode()
						curNode.KeyToChildNode[token] = newNode
						curNode = newNode
					} else if len(curNode.KeyToChildNode)+1 == d.MaxChildren {
						newNode := NewNode()
						curNode.KeyToChildNode[d.ParamStr] = newNode
						curNode = newNode
					} else {
						curNode = curNode.KeyToChildNode[d.ParamStr]
					}
				}
			}
		} else {
			curNode = curNode.KeyToChildNode[token]
		}

		currentDepth++
	}
}

// removeClusterFromTree removes a cluster from all leaf nodes in the prefix tree.
func (d *Drain) removeClusterFromTree(cluster *LogCluster) {
	d.removeClusterFromNode(d.RootNode, cluster.ClusterID)
}

func (d *Drain) removeClusterFromNode(node *Node, clusterID int) {
	// Remove from this node's cluster IDs
	newIDs := make([]int, 0, len(node.ClusterIDs))
	for _, id := range node.ClusterIDs {
		if id != clusterID {
			newIDs = append(newIDs, id)
		}
	}
	node.ClusterIDs = newIDs

	// Recursively remove from children
	for _, child := range node.KeyToChildNode {
		d.removeClusterFromNode(child, clusterID)
	}
}

// getClustersIDsForSeqLen returns all cluster IDs for templates with the given token count.
func (d *Drain) getClustersIDsForSeqLen(seqLen int) []int {
	tokenCountStr := strconv.Itoa(seqLen)
	node, ok := d.RootNode.KeyToChildNode[tokenCountStr]
	if !ok {
		return nil
	}
	return d.collectClusterIDs(node)
}

func (d *Drain) collectClusterIDs(node *Node) []int {
	var ids []int
	ids = append(ids, node.ClusterIDs...)
	for _, child := range node.KeyToChildNode {
		ids = append(ids, d.collectClusterIDs(child)...)
	}
	return ids
}

// putCluster adds a cluster to the store. Returns evicted cluster if any (LRU mode).
func (d *Drain) putCluster(cluster *LogCluster) *LogCluster {
	return d.IDToCluster.Put(cluster)
}

// Clusters returns all current log clusters.
func (d *Drain) Clusters() []*LogCluster {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.IDToCluster.Values()
}

// ClusterCount returns the number of active clusters.
func (d *Drain) ClusterCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.IDToCluster.Len()
}

// GetTotalClusterSize returns the sum of all cluster sizes (total messages processed).
func (d *Drain) GetTotalClusterSize() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	total := 0
	for _, c := range d.IDToCluster.Values() {
		total += c.Size
	}
	return total
}

// hasNumbers checks if a string contains any digit.
func hasNumbers(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// tokensEqual compares two token slices for equality.
func tokensEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// drainState is the JSON-serializable state of a Drain instance.
type drainState struct {
	RootNode        *Node         `json:"root_node"`
	IDToCluster     []*LogCluster `json:"id_to_cluster"`
	ClustersCounter int           `json:"clusters_counter"`
}

// MarshalJSON serializes the Drain state to JSON.
func (d *Drain) MarshalJSON() ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.marshalJSONUnlocked()
}

// marshalJSONUnlocked is the lock-free version. Caller must hold at least a read lock.
func (d *Drain) marshalJSONUnlocked() ([]byte, error) {
	state := drainState{
		RootNode:        d.RootNode,
		IDToCluster:     d.IDToCluster.Values(),
		ClustersCounter: d.ClustersCounter,
	}
	return json.Marshal(state)
}

// UnmarshalState restores the Drain state from JSON.
func (d *Drain) UnmarshalState(data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.unmarshalStateUnlocked(data)
}

// unmarshalStateUnlocked is the lock-free version. Caller must hold d.mu write lock.
func (d *Drain) unmarshalStateUnlocked(data []byte) error {

	var state drainState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	d.RootNode = state.RootNode
	d.ClustersCounter = state.ClustersCounter

	// Rebuild the cluster store
	var store ClusterStore
	if d.MaxClusters > 0 {
		store = NewLogClusterCache(d.MaxClusters)
	} else {
		store = newMapStore()
	}

	// Restore in reverse order: Values() is serialized MRU-first, but Put()
	// pushes to front, so inserting in reverse preserves the original LRU order.
	for i := len(state.IDToCluster) - 1; i >= 0; i-- {
		store.Put(state.IDToCluster[i])
	}
	d.IDToCluster = store

	return nil
}
