package lru

// EvictCallback is called when an item is evicted from the cache.
type EvictCallback[K comparable, V any] func(k K, v V)

type entry[K comparable, V any] struct {
	Key   K
	Value V
}

type historyEntry[K comparable, V any] struct {
	Key     K
	Value   V
	Visited uint
}

// CacheReplacing is the interface for simple LRU cache.
type CacheReplacing[K comparable, V any] interface {
	// Put a value to the cache, returns true if an eviction occurred and
	// updates the "recently used"-ness of the key.
	Put(key K, value V) (set, evicted bool)

	// Get returns key's value from the cache and
	// updates the "recently used"-ness of the key.
	Get(key K) (value V, ok bool)

	// Remove a key from the cache.
	Remove(key K) bool

	// Purge Clears all cache entries.
	Purge()
}
