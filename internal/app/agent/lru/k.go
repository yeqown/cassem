package lru

import (
	"container/list"
	"errors"
	"sync"
)

// K means lru-k. K is the type parameter for keys, V for values.
type LRU[K comparable, V any] struct {
	K       uint                // the K setting
	onEvict EvictCallback[K, V] // evict callback

	hMutex       sync.RWMutex
	hSize        uint                // historyMax - used = historyRest
	history      *list.List          // history doubly linked list
	historyItems map[K]*list.Element // history get op O(1)

	mutex      sync.RWMutex
	size       uint                // max - used = rest
	cache      *list.List          // cache doubly linked list
	cacheItems map[K]*list.Element // cache get op O(1)
}

// NewLRUK creates a new LRU-K cache.
func NewLRUK[K comparable, V any](k, size, hSize uint, onEvict EvictCallback[K, V]) (*LRU[K, V], error) {
	if k < 2 {
		return nil, errors.New("k is suggested bigger than 1, otherwise using LRU")
	}

	if hSize < size {
		hSize = size * ((size % 3) + 1)
	}

	return &LRU[K, V]{
		K:            k,
		onEvict:      onEvict,
		hMutex:       sync.RWMutex{},
		hSize:        hSize,
		history:      list.New(),
		historyItems: make(map[K]*list.Element),
		mutex:        sync.RWMutex{},
		size:         size,
		cache:        list.New(),
		cacheItems:   make(map[K]*list.Element),
	}, nil
}

// Put adds or updates a value in the cache.
func (c *LRU[K, V]) Put(key K, value V) (set, evicted bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if item, ok := c.cacheItems[key]; ok {
		item.Value.(*entry[K, V]).Value = value
		c.cache.MoveToFront(item)
		set = true
		return
	}

	var hEnt = &historyEntry[K, V]{}
	c.hMutex.Lock()
	defer c.hMutex.Unlock()
	item, ok := c.historyItems[key]
	if ok {
		hEnt = item.Value.(*historyEntry[K, V])
		hEnt.Visited++
		item.Value = hEnt
		if hEnt.Visited >= c.K {
			c.removeHistoryElement(item)

			e := &entry[K, V]{Key: key, Value: value}
			return true, c.addElement(e)
		}
		c.history.MoveToFront(item)
	} else {
		hEnt.Key = key
		hEnt.Value = value
		hEnt.Visited = 1
		item = c.addHistoryElement(hEnt)
	}

	return false, false
}

// Get returns key's value from the cache.
func (c *LRU[K, V]) Get(key K) (value V, ok bool) {
	c.mutex.Lock()
	if item, ok := c.cacheItems[key]; ok {
		c.cache.MoveToFront(item)
		c.mutex.Unlock()
		return item.Value.(*entry[K, V]).Value, true
	}
	c.mutex.Unlock()
	var zero V
	return zero, false
}

// Remove removes a key from the cache.
func (c *LRU[K, V]) Remove(key K) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if item, ok := c.cacheItems[key]; ok {
		c.removeElement(item)
		return true
	}
	return false
}

// Peek returns key's value without updating the recently used order.
func (c *LRU[K, V]) Peek(key K) (value V, ok bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var item *list.Element
	if item, ok = c.cacheItems[key]; ok {
		return item.Value.(*entry[K, V]).Value, true
	}
	var zero V
	return zero, ok
}

// Oldest returns the oldest entry in the cache.
func (c *LRU[K, V]) Oldest() (key K, value V, ok bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.cache == nil || c.cache.Len() == 0 {
		var zeroK K
		var zeroV V
		return zeroK, zeroV, false
	}

	item := c.cache.Back()
	ent := item.Value.(*entry[K, V])
	return ent.Key, ent.Value, true
}

// Keys returns all keys in the cache (oldest first).
func (c *LRU[K, V]) Keys() []K {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	keys := make([]K, len(c.cacheItems))
	i := 0
	for item := c.cache.Back(); item != nil; item = item.Prev() {
		keys[i] = item.Value.(*entry[K, V]).Key
		i++
	}
	return keys
}

// Len returns the number of items in the cache.
func (c *LRU[K, V]) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.cache == nil {
		return 0
	}
	return c.cache.Len()
}

// Purge clears all cache entries.
func (c *LRU[K, V]) Purge() {
	c.mutex.Lock()
	for k, v := range c.cacheItems {
		if c.onEvict != nil {
			c.onEvict(k, v.Value.(*entry[K, V]).Value)
		}
		delete(c.cacheItems, k)
	}
	c.cache.Init()
	c.mutex.Unlock()

	c.hMutex.Lock()
	for k := range c.historyItems {
		delete(c.historyItems, k)
	}
	c.history.Init()
	c.hMutex.Unlock()
}

func (c *LRU[K, V]) removeHistoryElement(item *list.Element) {
	c.hSize++
	ent := item.Value.(*historyEntry[K, V])
	c.history.Remove(item)
	delete(c.historyItems, ent.Key)
}

func (c *LRU[K, V]) addHistoryElement(hEnt *historyEntry[K, V]) *list.Element {
	if c.hSize == 0 {
		c.removeHistoryElement(c.history.Back())
	}
	c.hSize--
	c.historyItems[hEnt.Key] = c.history.PushFront(hEnt)
	return c.historyItems[hEnt.Key]
}

func (c *LRU[K, V]) removeElement(item *list.Element) {
	c.size++
	ent := item.Value.(*entry[K, V])
	c.cache.Remove(item)
	delete(c.cacheItems, ent.Key)
	if c.onEvict != nil {
		c.onEvict(ent.Key, ent.Value)
	}
}

func (c *LRU[K, V]) addElement(ent *entry[K, V]) (evicted bool) {
	if c.size == 0 {
		evicted = true
		c.removeElement(c.cache.Back())
	}
	c.size--
	c.cacheItems[ent.Key] = c.cache.PushFront(ent)
	return
}
