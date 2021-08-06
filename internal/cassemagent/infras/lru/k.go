package lru

import (
	"container/list"
	"errors"
	"sync"
)

var (
	_                CacheReplacing = &K{}
	historyEntryPool                = sync.Pool{
		New: func() interface{} {
			return new(historyEntry)
		},
	}
	entryPool = sync.Pool{
		New: func() interface{} {
			return new(entry)
		},
	}
)

// K . means lru-k
type K struct {
	K       uint          // the K setting
	onEvict EvictCallback // evict callback

	hMutex       sync.RWMutex
	hSize        uint                          // historyMax - used = historyRest
	history      *list.List                    // history doubly linked list
	historyItems map[interface{}]*list.Element // history get op O(1)

	mutex      sync.RWMutex
	size       uint                          // max - used = rest
	cache      *list.List                    // cache doubly linked list, save
	cacheItems map[interface{}]*list.Element // cache get op O(1)
}

// NewLRUK .
func NewLRUK(k, size, hSize uint, onEvict EvictCallback) (*K, error) {

	if k < 2 {
		return nil, errors.New("k is suggested bigger than 1, otherwise using LRU")
	}

	if hSize < size {
		hSize = size * ((size % 3) + 1) // why would I set this?
	}

	return &K{
		K:            k,
		onEvict:      onEvict,
		hMutex:       sync.RWMutex{},
		hSize:        hSize,
		history:      list.New(),
		historyItems: make(map[interface{}]*list.Element),
		mutex:        sync.RWMutex{},
		size:         size,
		cache:        list.New(),
		cacheItems:   make(map[interface{}]*list.Element),
	}, nil
}

// Put of K cache add or update
func (c *K) Put(key, value interface{}) (set, evicted bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if item, ok := c.cacheItems[key]; ok {
		item.Value.(*entry).Value = value
		c.cache.MoveToFront(item)
		set = true
		return
	}

	// if not hit in cache, then add to history
	var hEnt = historyEntryPool.Get().(*historyEntry)
	c.hMutex.Lock()
	defer c.hMutex.Unlock()
	item, ok := c.historyItems[key]
	if ok {
		hEnt = item.Value.(*historyEntry)
		// fmt.Printf("hit hEnt: %v\n", hEnt)
		hEnt = item.Value.(*historyEntry)
		hEnt.Visited++
		item.Value = hEnt
		if hEnt.Visited >= c.K {
			// true: move from history into cache
			c.removeHistoryElement(item)

			e := entryPool.Get().(*entry)
			e.Key = key
			e.Value = value
			return true, c.addElement(e)
		}
		// refresh history order
		c.history.MoveToFront(item)
	} else {
		// true: not exists
		hEnt.Key = key
		hEnt.Value = value
		hEnt.Visited = 1
		item = c.addHistoryElement(hEnt)
	}

	return false, false
}

// Get of K cache
func (c *K) Get(key interface{}) (value interface{}, ok bool) {
	c.mutex.Lock()
	// defer c.mutex.Unlock()
	// fmt.Println(c.cacheItems)
	if item, ok := c.cacheItems[key]; ok {
		c.cache.MoveToFront(item)
		c.mutex.Unlock()
		return item.Value.(*entry).Value, true
	}
	c.mutex.Unlock()
	return nil, false
}

// Remove of K cache
func (c *K) Remove(key interface{}) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if item, ok := c.cacheItems[key]; ok {
		c.removeElement(item)
		return true
	}
	return false
}

// Peek of K cache
func (c *K) Peek(key interface{}) (value interface{}, ok bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var item *list.Element
	if item, ok = c.cacheItems[key]; ok {
		return item.Value.(*entry).Value, true
	}
	return nil, ok
}

// Oldest of K cache
func (c *K) Oldest() (key, value interface{}, ok bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.cache == nil || c.cache.Len() == 0 {
		return nil, nil, false
	}

	item := c.cache.Back()
	ent := item.Value.(*entry)
	return ent.Value, ent.Value, true
}

// Keys of K cache
func (c *K) Keys() []interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	keys := make([]interface{}, len(c.cacheItems))
	i := 0
	for item := c.cache.Back(); item != nil; item = item.Prev() {
		keys[i] = item.Value.(*entry).Key
		i++
	}
	return keys
}

// Len of K cache
func (c *K) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.cache == nil {
		return 0
	}
	return c.cache.Len()
}

//// Iter of K cache
//func (c *K) Iter(f IterFunc) {
//	c.mutex.RLock()
//	defer c.mutex.RUnlock()
//	for item := c.cache.Back(); item != nil; item = item.Prev() {
//		ent := item.Value.(*entry)
//		f(ent.Key, ent.Value)
//	}
//}

// Purge of K cache
func (c *K) Purge() {
	c.mutex.Lock()
	for k, v := range c.cacheItems {
		if c.onEvict != nil {
			c.onEvict(k, v.Value.(*entry).Value)
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

func (c *K) removeHistoryElement(item *list.Element) {
	c.hSize++
	ent := item.Value.(*historyEntry)
	historyEntryPool.Put(ent)
	c.history.Remove(item)
	delete(c.historyItems, ent.Key)
}

func (c *K) addHistoryElement(hEnt *historyEntry) *list.Element {
	if c.hSize == 0 {
		c.removeHistoryElement(c.history.Back())
	}
	c.hSize--
	// item := c.history.PushFront(hEnt)
	// c.historyItems[hEnt.Key] = item
	// return item
	c.historyItems[hEnt.Key] = c.history.PushFront(hEnt)
	return c.historyItems[hEnt.Key]
}

func (c *K) removeElement(item *list.Element) {
	c.size++
	ent := item.Value.(*entry)
	entryPool.Put(ent)
	c.cache.Remove(item)
	delete(c.cacheItems, ent.Key)
	if c.onEvict != nil {
		c.onEvict(ent.Key, ent.Value)
	}
}

func (c *K) addElement(ent *entry) (evicted bool) {
	// println(c.size)
	if c.size == 0 {
		evicted = true
		c.removeElement(c.cache.Back())
	}
	c.size--
	c.cacheItems[ent.Key] = c.cache.PushFront(ent)
	return
}
