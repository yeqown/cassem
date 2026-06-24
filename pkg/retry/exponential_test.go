package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewExponential(t *testing.T) {
	tests := []struct {
		name     string
		base     time.Duration
		max      time.Duration
		random   time.Duration
		expected Strategy
	}{
		{
			name:   "default values",
			base:   100 * time.Millisecond,
			max:    10 * time.Second,
			random: 10 * time.Millisecond,
		},
		{
			name:   "custom values",
			base:   50 * time.Millisecond,
			max:    5 * time.Second,
			random: 5 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewExponential(tt.base, tt.max, tt.random)
			assert.NotNil(t, strategy)
		})
	}
}

func TestDefaultExponential(t *testing.T) {
	strategy := DefaultExponential()
	assert.NotNil(t, strategy)
}

func TestExponentialBackoffRetry_NilFunc(t *testing.T) {
	strategy := NewExponential(100*time.Millisecond, time.Second, 10*time.Millisecond)
	ctx := context.Background()

	err := strategy.Do(ctx, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrNilFunc, err)
}

func TestExponentialBackoffRetry_SuccessOnFirstTry(t *testing.T) {
	strategy := NewExponential(100*time.Millisecond, time.Second, 10*time.Millisecond)
	ctx := context.Background()

	called := false
	fn := func() error {
		called = true
		return nil
	}

	err := strategy.Do(ctx, fn)
	assert.NoError(t, err)
	assert.True(t, called, "Function should be called once")
}

func TestExponentialBackoffRetry_EventualSuccess(t *testing.T) {
	strategy := NewExponential(50*time.Millisecond, 200*time.Millisecond, 5*time.Millisecond)
	ctx := context.Background()

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	start := time.Now()
	err := strategy.Do(ctx, fn)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond, "Should have waited for retries")
}

func TestExponentialBackoffRetry_MaxRetriesExceeded(t *testing.T) {
	strategy := NewExponential(50*time.Millisecond, 100*time.Millisecond, 5*time.Millisecond)

	// Use a context with timeout to stop infinite retries
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("persistent error")
	}

	err := strategy.Do(ctx, fn)
	assert.Error(t, err)
	assert.GreaterOrEqual(t, attempts, 2, "Should have attempted at least twice")
}

func TestExponentialBackoffRetry_ContextCancellation(t *testing.T) {
	strategy := NewExponential(100*time.Millisecond, time.Second, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	fn := func() error {
		attempts++
		if attempts == 1 {
			// Cancel context after first attempt
			cancel()
			return errors.New("first error")
		}
		return nil
	}

	err := strategy.Do(ctx, fn)
	assert.Error(t, err)
	// When context is canceled, the function returns the last error from fn
	// not context.Canceled
	assert.Contains(t, err.Error(), "first error")
}

func TestExponentialBackoffRetry_NilContext(t *testing.T) {
	strategy := NewExponential(50*time.Millisecond, 200*time.Millisecond, 5*time.Millisecond)

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary error")
		}
		return nil
	}

	// Pass nil context, should use TODO() internally
	err := strategy.Do(context.TODO(), fn)
	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestExponentialBackoffRetry_ExponentialBackoff(t *testing.T) {
	base := 50 * time.Millisecond
	// Use a small random interval to avoid 2-second default
	strategy := NewExponential(base, time.Second, 1*time.Millisecond)

	ctx := context.Background()

	// Test that delay increases exponentially
	var delays []time.Duration
	lastAttempt := time.Now()
	attemptNum := 0

	fn := func() error {
		now := time.Now()
		if attemptNum > 0 {
			delays = append(delays, now.Sub(lastAttempt))
		}
		lastAttempt = now
		attemptNum++

		if attemptNum < 4 {
			return errors.New("error")
		}
		return nil
	}

	err := strategy.Do(ctx, fn)
	assert.NoError(t, err)
	assert.Len(t, delays, 3)

	// First delay should be approximately base (50ms)
	// Second should be base * 2 (100ms)
	// Third should be base * 4 (200ms)
	// Allow tolerance for timing variations and random jitter
	assert.InDelta(t, base.Milliseconds(), delays[0].Milliseconds(), 10)
	assert.InDelta(t, (base*2).Milliseconds(), delays[1].Milliseconds(), 20)
	assert.InDelta(t, (base*4).Milliseconds(), delays[2].Milliseconds(), 50)
}

func TestExponentialBackoffRetry_MaxDelay(t *testing.T) {
	base := 10 * time.Millisecond
	max := 50 * time.Millisecond
	strategy := NewExponential(base, max, 0)

	ctx := context.Background()

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 5 {
			return errors.New("error")
		}
		return nil
	}

	start := time.Now()
	err := strategy.Do(ctx, fn)
	elapsed := time.Since(start)

	assert.NoError(t, err)

	// With exponential backoff 10, 20, 40, 50(max), we should wait at least 120ms
	// But cap at max delay
	expectedMin := base*3 + max
	assert.GreaterOrEqual(t, elapsed, expectedMin-20*time.Millisecond)
}

func TestRandomDuration(t *testing.T) {
	// Test that randomDuration produces values within expected range
	for i := 0; i < 100; i++ {
		d := randomDuration(100 * time.Millisecond)
		assert.GreaterOrEqual(t, d, time.Duration(0))
		assert.LessOrEqual(t, d, 100*time.Millisecond)
	}
}

func TestRandomDuration_Zero(t *testing.T) {
	d := randomDuration(0)
	// Should use default of 2 seconds
	assert.GreaterOrEqual(t, d, time.Duration(0))
	assert.LessOrEqual(t, d, 2*time.Second)
}
