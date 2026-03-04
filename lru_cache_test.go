package drain3

import (
	"testing"
)

func TestLogClusterCacheNonDestructiveGet(t *testing.T) {
	cache := NewLogClusterCache(3)

	c1 := NewLogCluster([]string{"a"}, 1)
	c2 := NewLogCluster([]string{"b"}, 2)
	c3 := NewLogCluster([]string{"c"}, 3)

	cache.Put(c1) // order: 1
	cache.Put(c2) // order: 2, 1
	cache.Put(c3) // order: 3, 2, 1

	// Get c1 — should NOT move it to front (non-destructive)
	got := cache.Get(1)
	if got != c1 {
		t.Fatalf("expected c1, got %v", got)
	}

	// Now insert c4 — should evict c1 (LRU), NOT c2
	c4 := NewLogCluster([]string{"d"}, 4)
	evicted := cache.Put(c4)
	if evicted == nil || evicted.ClusterID != 1 {
		t.Fatalf("expected eviction of cluster 1, got %v", evicted)
	}

	if cache.Get(1) != nil {
		t.Fatal("cluster 1 should have been evicted")
	}
}

func TestLogClusterCacheTouch(t *testing.T) {
	cache := NewLogClusterCache(3)

	c1 := NewLogCluster([]string{"a"}, 1)
	c2 := NewLogCluster([]string{"b"}, 2)
	c3 := NewLogCluster([]string{"c"}, 3)

	cache.Put(c1) // order: 1
	cache.Put(c2) // order: 2, 1
	cache.Put(c3) // order: 3, 2, 1

	// Touch c1 — should move it to front
	cache.Touch(1)
	// Now order should be: 1, 3, 2

	c4 := NewLogCluster([]string{"d"}, 4)
	evicted := cache.Put(c4)
	// c2 should be evicted (LRU after touch)
	if evicted == nil || evicted.ClusterID != 2 {
		t.Fatalf("expected eviction of cluster 2, got %v", evicted)
	}
}

func TestLogClusterCacheOverwrite(t *testing.T) {
	cache := NewLogClusterCache(3)

	c1 := NewLogCluster([]string{"a"}, 1)
	cache.Put(c1)

	// Overwrite with updated cluster
	c1Updated := NewLogCluster([]string{"a", "b"}, 1)
	c1Updated.Size = 5
	evicted := cache.Put(c1Updated)
	if evicted != nil {
		t.Fatalf("no eviction expected on overwrite, got %v", evicted)
	}

	got := cache.Get(1)
	if got.Size != 5 {
		t.Fatalf("expected size 5, got %d", got.Size)
	}
	if cache.Len() != 1 {
		t.Fatalf("expected len 1, got %d", cache.Len())
	}
}

func TestLogClusterCacheEviction(t *testing.T) {
	cache := NewLogClusterCache(2)

	c1 := NewLogCluster([]string{"a"}, 1)
	c2 := NewLogCluster([]string{"b"}, 2)
	c3 := NewLogCluster([]string{"c"}, 3)

	cache.Put(c1)
	cache.Put(c2)
	evicted := cache.Put(c3)

	if evicted == nil || evicted.ClusterID != 1 {
		t.Fatalf("expected eviction of cluster 1, got %v", evicted)
	}
	if cache.Len() != 2 {
		t.Fatalf("expected len 2, got %d", cache.Len())
	}
}

func TestLogClusterCacheValues(t *testing.T) {
	cache := NewLogClusterCache(5)

	for i := 1; i <= 3; i++ {
		cache.Put(NewLogCluster([]string{"x"}, i))
	}

	vals := cache.Values()
	if len(vals) != 3 {
		t.Fatalf("expected 3 values, got %d", len(vals))
	}
	// Most recently used first
	if vals[0].ClusterID != 3 || vals[1].ClusterID != 2 || vals[2].ClusterID != 1 {
		t.Fatalf("unexpected order: %v", vals)
	}
}

func TestLogClusterCacheRemove(t *testing.T) {
	cache := NewLogClusterCache(5)
	c1 := NewLogCluster([]string{"a"}, 1)
	cache.Put(c1)

	cache.Remove(1)
	if cache.Get(1) != nil {
		t.Fatal("expected nil after removal")
	}
	if cache.Len() != 0 {
		t.Fatalf("expected len 0, got %d", cache.Len())
	}
}

func TestMapStore(t *testing.T) {
	store := newMapStore()

	c1 := NewLogCluster([]string{"a"}, 1)
	c2 := NewLogCluster([]string{"b"}, 2)

	store.Put(c1)
	store.Put(c2)

	if store.Len() != 2 {
		t.Fatalf("expected 2, got %d", store.Len())
	}

	got := store.Get(1)
	if got != c1 {
		t.Fatalf("expected c1, got %v", got)
	}

	store.Remove(1)
	if store.Get(1) != nil {
		t.Fatal("expected nil after removal")
	}

	vals := store.Values()
	if len(vals) != 1 {
		t.Fatalf("expected 1 value, got %d", len(vals))
	}
}
