package benchmark_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yeqown/cassem/internal/cassemagent/infras/lru"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/internal/cassemdb/infras/storage"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/log"
)

func TestMain(m *testing.M) {
	log.SetLogLevel(log.LevelError)
	os.Exit(m.Run())
}

func BenchmarkLRUKPut100Capacity100History(b *testing.B) {
	benchmarkLRUKPut(b, 100, 100)
}

func BenchmarkLRUKPut50Capacity100History(b *testing.B) {
	benchmarkLRUKPut(b, 50, 100)
}

func benchmarkLRUKPut(b *testing.B, size int, historySize int) {
	b.Helper()
	cache, err := lru.NewLRUK[int, int](2, uint(size), uint(historySize), nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := i % historySize
		cache.Put(key, key)
	}
}

func BenchmarkStorageRepositoryWrite(b *testing.B) {
	cases := []struct {
		name string
		size int
	}{
		{name: "32B", size: 32},
		{name: "1KB", size: 1024},
		{name: "10KB", size: 10 * 1024},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			dir := b.TempDir()
			repo, err := storage.NewRepository(&conf.Bolt{Dir: dir, DB: "benchmark.db"})
			if err != nil {
				b.Fatal(err)
			}

			payload := []byte(strings.Repeat("a", tc.size))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("benchmark/write/%s/%d", tc.name, i)
				entity := apicassemdb.NewEntityWithCreated(key, payload, 30, time.Now().Unix())
				if err = repo.SetKV(key, entity, false); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkStorageRepositoryRange(b *testing.B) {
	dir := b.TempDir()
	repo, err := storage.NewRepository(&conf.Bolt{Dir: dir, DB: "benchmark.db"})
	if err != nil {
		b.Fatal(err)
	}

	payload := []byte(strings.Repeat("a", 128))
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("benchmark/range/%04d", i)
		entity := apicassemdb.NewEntityWithCreated(key, payload, 30, time.Now().Unix())
		if err = repo.SetKV(key, entity, false); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := repo.Range("benchmark/range", "", 100)
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Items) != 100 {
			b.Fatalf("expected 100 items, got %d", len(result.Items))
		}
	}
}
