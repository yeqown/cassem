package runtime

import (
	"os"
	"sync"
)

var (
	_debug     bool
	_debugOnce sync.Once
)

func IsDebug() bool {
	_debugOnce.Do(func() {
		mapping := map[string]struct{}{
			"1":    {},
			"TRUE": {},
			"true": {},
		}

		v := os.Getenv("DEBUG")
		if _, ok := mapping[v]; ok {
			_debug = true
		}
	})

	return _debug
}
