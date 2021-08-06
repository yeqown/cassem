package lru

// EvictCallback .
type EvictCallback func(k, v interface{})

//// IterFunc .
//type IterFunc func(k, v interface{})

type entry struct {
	Key   interface{}
	Value interface{}
}

type historyEntry struct {
	Key     interface{}
	Value   interface{}
	Visited uint
}

// CacheReplacing is the interface for simple LRU cache.
type CacheReplacing interface {
	// Put a value to the cache, returns true if an eviction occurred and
	// updates the "recently used"-ness of the key.
	Put(key, value interface{}) (set, evicted bool)

	// Get returns key's value from the cache and
	// updates the "recently used"-ness of the key. #value, isFound
	Get(key interface{}) (value interface{}, ok bool)

	// Remove a key from the cache.
	Remove(key interface{}) bool

	// Purge Clears all cache entries.
	Purge()
}
