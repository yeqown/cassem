package cache

import (
	"encoding/json"
	"sync"
)

type nonCache struct {
	sync.RWMutex

	data map[string][]byte
}

func (n *nonCache) Persist() ([]byte, error) {
	n.RLock()
	data := n.data
	n.RUnlock()

	return json.Marshal(data)
}

func (n *nonCache) Restore(data []byte) error {
	n.Lock()
	defer n.Unlock()

	// clear all data
	n.data = make(map[string][]byte)

	return json.Unmarshal(data, &(n.data))
}

func (n *nonCache) Set(key string, v []byte) SetResult {
	n.Lock()
	defer n.Unlock()

	n.data[key] = v
	return SetResult{
		err:           nil,
		NeedSync:      true,
		NeedDeleteKey: "",
	}
}

func (n *nonCache) Get(key string) ([]byte, error) {
	n.RLock()
	defer n.RUnlock()

	v, ok := n.data[key]
	if !ok {

		return nil, ErrMiss
	}

	return v, nil
}

func (n *nonCache) Del(key string) error {
	n.Lock()
	defer n.Unlock()

	delete(n.data, key)

	return nil
}

func NewNonCache() ICache {
	return &nonCache{
		RWMutex: sync.RWMutex{},
		data:    make(map[string][]byte),
	}
}
