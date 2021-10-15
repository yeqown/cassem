package retry

import (
	"context"
	"math/rand"
	"time"

	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/pkg/errors"
)

var ErrNilFunc = errors.New("retry: func is nil")

// exponentialBackoffRetry 指数回退重试策略
// DONE(@yeqiang): 注意这里不能有状态，测试时考虑并发场景
type exponentialBackoffRetry struct {
	base     time.Duration
	maxDelay time.Duration

	randomInterval time.Duration
}

func NewExponential(base, max, rand time.Duration) Strategy {
	return exponentialBackoffRetry{
		base:           base,
		maxDelay:       max,
		randomInterval: rand,
	}
}

func DefaultExponential() Strategy {
	return NewExponential(
		time.Millisecond*100,
		time.Second*10,
		time.Millisecond*10,
	)
}

func (e exponentialBackoffRetry) Do(ctx context.Context, fn func() error) (err error) {
	if fn == nil {
		return ErrNilFunc
	}

	if ctx == nil {
		ctx = context.TODO()
	}

	n := uint(0)
	next := e.base

retry:
	for {
		if err = fn(); err == nil {
			break
		}

		if next < e.maxDelay {
			next = e.base * (1 << n)

			if next > e.maxDelay {
				next = e.maxDelay
			}
		}
		// 增加抖动，避免同一时间大量重试
		r := randomDuration(e.randomInterval)
		t := time.NewTimer(next + r)

		if runtime.IsDebug() {
			println((next + r).Milliseconds())
		}

		select {
		case <-ctx.Done():
			break retry
		case <-t.C:
			// do nothing, just wait the timer
		}

		// 重试次数 +1
		n++
	}

	return
}

const (
	_two_seconds = 2 * time.Second
)

func randomDuration(max time.Duration) time.Duration {
	if max == 0 {
		max = _two_seconds
	}
	return time.Duration(rand.Int63n(int64(max)))
}
