package lru_test

import (
	"fmt"
	"testing"

	"github.com/yeqown/cassem/internal/cassemagent/infras/lru"
)

func Test_LRUK(t *testing.T) {
	cache, err := lru.NewLRUK(2, 2, 4, func(k, v interface{}) {
		fmt.Printf("onEvict: k: %v, v: %v\n", k, v)
	})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	cache.Put(2, 1)
	if _, hit := cache.Get(2); hit {
		t.Error("should not get 2 hit")
	}
	cache.Put(3, 1)
	if _, hit := cache.Get(3); hit {
		t.Error("should not get 3 hit")
	}
	cache.Put(3, 1)
	if _, hit := cache.Get(3); !hit {
		t.Error("should get 3 hit")
	}
	cache.Put(4, 1)
	cache.Put(4, 1)
	if _, hit := cache.Get(2); hit {
		t.Error("could not get 2 hit")
	}

	// update cache
	cache.Put(4, 2)
	if v, hit := cache.Get(4); !hit || v.(int) != 2 {
		t.Error("should get 4 hit and value shoult be 2")
	}
	if _, hit := cache.Get(3); !hit {
		t.Error("should get 3 hit")
	}

	// trigger replacing
	cache.Put(5, 1)
	cache.Put(5, 1)
	println("cur length: ", cache.Len())
	if _, hit := cache.Get(4); hit {
		t.Error("should not get 4 hit")
	}
	if _, hit := cache.Get(5); !hit {
		t.Error("should get 5 hit")
	}
}

func Benchmark_LRUK_100_100(b *testing.B) {
	cache, err := lru.NewLRUK(2, 100, 100, nil)
	// size: 50
	// hSize: 100
	if err != nil {
		b.Error(err)
		b.FailNow()
	}

	for i := 0; i < b.N; i++ {
		key := i % 100
		cache.Put(key, key)
	}
}

// Benchmark_LRUK-4   	10000000	       159 ns/op	      15 B/op	       1 allocs/op

func Benchmark_LRUK_50_100(b *testing.B) {
	cache, err := lru.NewLRUK(2, 50, 100, nil)
	// size: 50
	// hSize: 100
	if err != nil {
		b.Error(err)
		b.FailNow()
	}

	for i := 0; i < b.N; i++ {
		key := i % 100
		cache.Put(key, key)
	}
}

// Benchmark_LRUK_50_100-4   	 3000000	       536 ns/op	      90 B/op	       3 allocs/op
// To analyze:
//
// go test -bench=. -benchmem -memprofile memprofile.out -cpuprofile profile.out
//
//
