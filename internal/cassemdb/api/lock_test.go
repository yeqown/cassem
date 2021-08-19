package api

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_lock(t *testing.T) {
	conn, err := DialWithMode([]string{"127.0.0.1:2021", "127.0.0.1:2022", "127.0.0.1:2023"}, Mode_X)
	assert.NoError(t, err)

	kv := NewKVClient(conn)
	wg := sync.WaitGroup{}

	go func() {
		wg.Add(1)
		defer wg.Done()
		assert.NotPanics(t, func() {
			WithLock(kv, "locks/Test_lock", 10, func() {
				time.Sleep(2 * time.Second)
			})
		})
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		assert.Panics(t, func() {
			WithLock(kv, "locks/Test_lock", 10, func() {
				time.Sleep(2 * time.Second)
			})
		})
	}()

	wg.Wait()
}
