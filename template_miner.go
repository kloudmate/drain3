package drain3

import (
	"bytes"
	"compress/zlib"
	"io"
	"strconv"
	"time"
)

// AddLogMessageResult is the result of processing a log message.
type AddLogMessageResult struct {
	Cluster    *LogCluster
	ChangeType ChangeType
}

// TemplateMiner is the high-level API that integrates Drain, masking, persistence, and profiling.
// Thread safety is provided by Drain's internal RWMutex — no separate lock is needed.
type TemplateMiner struct {
	Drain       *Drain
	Config      *Config
	Persistence PersistenceHandler
	Profiler    Profiler
	Masker      *LogMasker

	lastSaveTime time.Time
}

// NewTemplateMiner creates a new TemplateMiner with the given persistence handler and config.
// If config is nil, DefaultConfig() is used. If persistence is nil, no persistence is performed.
func NewTemplateMiner(persistence PersistenceHandler, config *Config) (*TemplateMiner, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Derive param_str from mask prefix/suffix like Python does
	paramStr := config.Drain.ParamStr
	if paramStr == "" || paramStr == DefaultParamStr {
		paramStr = config.Masking.MaskPrefix + "*" + config.Masking.MaskSuffix
	}

	drainCfg := DrainConfig{
		SimTh:                    config.Drain.SimTh,
		Depth:                    config.Drain.Depth,
		MaxChildren:              config.Drain.MaxChildren,
		MaxClusters:              config.Drain.MaxClusters,
		ExtraDelimiters:          config.Drain.ExtraDelimiters,
		ParamStr:                 paramStr,
		ParametrizeNumericTokens: config.Drain.GetParametrizeNumericTokens(),
	}

	drain := NewDrain(drainCfg)

	// Build masking instructions
	var instructions []*MaskingInstruction
	for _, ic := range config.Masking.Instructions {
		inst, err := NewMaskingInstruction(ic.Pattern, ic.MaskWith)
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, inst)
	}

	masker := NewLogMasker(instructions, config.Masking.MaskPrefix, config.Masking.MaskSuffix)

	var profiler Profiler
	if config.Profiling.Enabled {
		profiler = NewSimpleProfiler()
	} else {
		profiler = NullProfiler{}
	}

	tm := &TemplateMiner{
		Drain:        drain,
		Config:       config,
		Persistence:  persistence,
		Profiler:     profiler,
		Masker:       masker,
		lastSaveTime: time.Now(),
	}

	// Load saved state if available
	if persistence != nil {
		if err := tm.LoadState(); err != nil {
			// Non-fatal: just start fresh
			_ = err
		}
	}

	return tm, nil
}

// AddLogMessage processes a log message through the masking and Drain pipeline.
func (tm *TemplateMiner) AddLogMessage(message string) *AddLogMessageResult {
	tm.Drain.mu.Lock()
	defer tm.Drain.mu.Unlock()

	tm.Profiler.StartSection("masking")
	maskedContent := tm.Masker.Mask(message)
	tm.Profiler.EndSection("masking")

	tm.Profiler.StartSection("drain")
	cluster, changeType := tm.Drain.addLogMessageUnlocked(maskedContent)
	tm.Profiler.EndSection("drain")

	// Auto-save: on any change, or periodically
	if tm.Persistence != nil {
		snapshotReason := tm.getSnapshotReason(changeType, cluster.ClusterID)
		if snapshotReason != "" {
			tm.Profiler.StartSection("save_state")
			_ = tm.saveStateInternal()
			tm.Profiler.EndSection("save_state")
		}
	}

	return &AddLogMessageResult{
		Cluster:    cluster,
		ChangeType: changeType,
	}
}

// Match finds the best matching cluster for a log message without modifying state.
func (tm *TemplateMiner) Match(message string, strategy SearchStrategy) *LogCluster {
	tm.Drain.mu.RLock()
	defer tm.Drain.mu.RUnlock()

	maskedContent := tm.Masker.Mask(message)
	return tm.Drain.matchUnlocked(maskedContent, strategy)
}

// getSnapshotReason returns a non-empty reason if state should be saved, matching Python's logic.
// Saves on any change, or periodically if the interval has elapsed.
func (tm *TemplateMiner) getSnapshotReason(changeType ChangeType, clusterID int) string {
	if changeType != ChangeNone {
		return changeType.String() + " (" + strconv.Itoa(clusterID) + ")"
	}
	if time.Since(tm.lastSaveTime) >= time.Duration(tm.Config.Snapshot.SnapshotIntervalMinutes)*time.Minute {
		return "periodic"
	}
	return ""
}

// SaveState persists the current Drain state.
func (tm *TemplateMiner) SaveState() error {
	tm.Drain.mu.Lock()
	defer tm.Drain.mu.Unlock()
	return tm.saveStateInternal()
}

// saveStateInternal persists state. Caller must hold tm.Drain.mu.
func (tm *TemplateMiner) saveStateInternal() error {
	if tm.Persistence == nil {
		return nil
	}

	data, err := tm.Drain.marshalJSONUnlocked()
	if err != nil {
		return err
	}

	if tm.Config.Snapshot.CompressState {
		var buf bytes.Buffer
		w := zlib.NewWriter(&buf)
		if _, err := w.Write(data); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		data = buf.Bytes()
	}

	if err := tm.Persistence.SaveState(data); err != nil {
		return err
	}
	tm.lastSaveTime = time.Now()
	return nil
}

// LoadState restores Drain state from persistence.
func (tm *TemplateMiner) LoadState() error {
	if tm.Persistence == nil {
		return nil
	}

	data, err := tm.Persistence.LoadState()
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	// Try to decompress (zlib)
	if tm.Config.Snapshot.CompressState {
		r, err := zlib.NewReader(bytes.NewReader(data))
		if err == nil {
			decompressed, err := io.ReadAll(r)
			r.Close()
			if err == nil {
				data = decompressed
			}
		}
	}

	tm.Drain.mu.Lock()
	defer tm.Drain.mu.Unlock()
	return tm.Drain.unmarshalStateUnlocked(data)
}

// Clusters returns all current log clusters.
func (tm *TemplateMiner) Clusters() []*LogCluster {
	return tm.Drain.Clusters()
}

// ClusterCount returns the number of active clusters.
func (tm *TemplateMiner) ClusterCount() int {
	return tm.Drain.ClusterCount()
}

// GetProfilerReport returns the profiler report string.
func (tm *TemplateMiner) GetProfilerReport(reset bool) string {
	return tm.Profiler.Report(reset)
}
