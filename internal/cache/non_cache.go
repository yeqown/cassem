package cache

type nonCache struct {
	data map[string][]byte
}

func (n nonCache) Set(key string, v []byte) (evicted bool, err error) {
	n.data[key] = v
	return false, nil
}

func (n nonCache) Get(key string) ([]byte, error) {
	v, ok := n.data[key]
	if !ok {
		return nil, ErrMiss
	}

	return v, nil
}

func newNonCache() *nonCache {
	return &nonCache{
		data: make(map[string][]byte),
	}
}
