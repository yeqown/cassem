package api

import (
	"context"
	"time"

	"github.com/yeqown/log"
)

type distributedLock struct {
	key string
	ttl int
}

func newLock(key string, ttl int) distributedLock {
	return distributedLock{
		key: key,
		ttl: ttl,
	}
}

func (l distributedLock) Acquire(kv KVClient) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if l.ttl <= 0 {
		l.ttl = 10
	}
	if l.ttl > 2*24*3600 {
		l.ttl = 2 * 24 * 3600
	}

	if _, err = kv.SetKV(ctx, &SetKVReq{
		Key:       l.key,
		IsDir:     false,
		Ttl:       0,
		Val:       nil,
		Overwrite: false,
	}); err != nil {
		return
	}

	return
}

func (l distributedLock) Release(kv KVClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := kv.UnsetKV(ctx, &UnsetKVReq{
		Key:   l.key,
		IsDir: false,
	})

	if err != nil {
		log.WithFields(log.Fields{"error": err, "key": l.key}).
			Error("distributedLock failed to release lock")
	}

	return err
}

func WithLock(kv KVClient, lockKey string, ttl int, f func()) {
	lock := newLock(lockKey, ttl)
	if err := lock.Acquire(kv); err != nil {
		panic(err)
	}

	f()

	if err := lock.Release(kv); err != nil {
		panic(err)
	}
}
