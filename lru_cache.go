package drain3

import (
	"container/list"
)

// ClusterStore is an interface for storing and retrieving log clusters.
type ClusterStore interface {
	Get(clusterID int) *LogCluster
	// Put inserts or updates a cluster. Returns the evicted cluster if any (LRU mode).
	Put(cluster *LogCluster) *LogCluster
	Remove(clusterID int)
	Len() int
	Values() []*LogCluster
}

// mapStore is an unlimited-capacity cluster store backed by a map.
type mapStore struct {
	m map[int]*LogCluster
}

func newMapStore() *mapStore {
	return &mapStore{m: make(map[int]*LogCluster)}
}

func (s *mapStore) Get(clusterID int) *LogCluster {
	return s.m[clusterID]
}

func (s *mapStore) Put(cluster *LogCluster) *LogCluster {
	s.m[cluster.ClusterID] = cluster
	return nil
}

func (s *mapStore) Remove(clusterID int) {
	delete(s.m, clusterID)
}

func (s *mapStore) Len() int {
	return len(s.m)
}

func (s *mapStore) Values() []*LogCluster {
	result := make([]*LogCluster, 0, len(s.m))
	for _, v := range s.m {
		result = append(result, v)
	}
	return result
}

// LogClusterCache is an LRU cache for log clusters.
// IMPORTANT: Get() does NOT promote the entry (non-destructive).
// Only Put() and Touch() move entries to the front (most recently used).
// This matches the Python Drain3 behavior where reading a cluster for matching
// should not affect eviction order.
type LogClusterCache struct {
	maxSize int
	items   map[int]*list.Element
	order   *list.List // front = most recently used
}

type lruEntry struct {
	cluster *LogCluster
}

// NewLogClusterCache creates a new LRU cache with the given maximum size.
func NewLogClusterCache(maxSize int) *LogClusterCache {
	return &LogClusterCache{
		maxSize: maxSize,
		items:   make(map[int]*list.Element),
		order:   list.New(),
	}
}

// Get retrieves a cluster by ID without affecting eviction order.
func (c *LogClusterCache) Get(clusterID int) *LogCluster {
	if elem, ok := c.items[clusterID]; ok {
		return elem.Value.(*lruEntry).cluster
	}
	return nil
}

// Put inserts or updates a cluster, promoting it to most-recently-used.
// If the cache is at capacity, the least-recently-used entry is evicted.
// Returns the evicted cluster, if any.
func (c *LogClusterCache) Put(cluster *LogCluster) *LogCluster {
	if elem, ok := c.items[cluster.ClusterID]; ok {
		// Update existing entry and move to front
		elem.Value.(*lruEntry).cluster = cluster
		c.order.MoveToFront(elem)
		return nil
	}

	var evicted *LogCluster
	// Evict if at capacity
	if c.maxSize > 0 && c.order.Len() >= c.maxSize {
		evicted = c.evictOldest()
	}

	// Insert new entry at front
	entry := &lruEntry{cluster: cluster}
	elem := c.order.PushFront(entry)
	c.items[cluster.ClusterID] = elem
	return evicted
}

// Touch promotes a cluster to most-recently-used without modifying it.
func (c *LogClusterCache) Touch(clusterID int) {
	if elem, ok := c.items[clusterID]; ok {
		c.order.MoveToFront(elem)
	}
}

// Remove removes a cluster from the cache.
func (c *LogClusterCache) Remove(clusterID int) {
	if elem, ok := c.items[clusterID]; ok {
		c.order.Remove(elem)
		delete(c.items, clusterID)
	}
}

// Len returns the number of clusters in the cache.
func (c *LogClusterCache) Len() int {
	return c.order.Len()
}

// Values returns all clusters in the cache (most recently used first).
func (c *LogClusterCache) Values() []*LogCluster {
	result := make([]*LogCluster, 0, c.order.Len())
	for e := c.order.Front(); e != nil; e = e.Next() {
		result = append(result, e.Value.(*lruEntry).cluster)
	}
	return result
}

func (c *LogClusterCache) evictOldest() *LogCluster {
	back := c.order.Back()
	if back == nil {
		return nil
	}
	entry := back.Value.(*lruEntry)
	c.order.Remove(back)
	delete(c.items, entry.cluster.ClusterID)
	return entry.cluster
}
