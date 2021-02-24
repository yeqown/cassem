package set

type StringSet map[string]struct{}

func NewStringSet(size int) StringSet {
	if size <= 0 {
		size = 8
	}

	return make(StringSet, size)
}

func (s StringSet) Add(key string) (evicted bool) {
	_, evicted = s[key]
	if !evicted {
		s[key] = struct{}{}
	}
	return evicted
}

func (s StringSet) Adds(keys []string) {
	for _, key := range keys {
		s[key] = struct{}{}
	}
}

func (s StringSet) Keys() []string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}

	return keys
}
