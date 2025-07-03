package limiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fakeTime struct {
	current time.Time
}

func (f *fakeTime) Now() time.Time {
	return f.current
}

func (f *fakeTime) Advance(d time.Duration) {
	f.current = f.current.Add(d)
}

func TestBurstRateLimiter_Allow_Burst(t *testing.T) {
	ft := &fakeTime{current: time.Unix(0, 0)}
	logger := zaptest.NewLogger(t)
	limiter := NewBurstRateLimiterWithTimeProvider(2, 3, logger, ft.Now)

	// Should allow up to burst size immediately
	require.True(t, limiter.Allow())
	require.True(t, limiter.Allow())
	require.True(t, limiter.Allow())
	assert.False(t, limiter.Allow(), "should not allow more than burst")
}

func TestBurstRateLimiter_Refill(t *testing.T) {
	ft := &fakeTime{current: time.Unix(0, 0)}
	logger := zaptest.NewLogger(t)
	limiter := NewBurstRateLimiterWithTimeProvider(2, 3, logger, ft.Now)

	// Deplete tokens
	require.True(t, limiter.Allow())
	require.True(t, limiter.Allow())
	require.True(t, limiter.Allow())
	assert.False(t, limiter.Allow())

	// Advance time to refill 2 tokens (1 second at 2 rps)
	ft.Advance(time.Second)
	assert.True(t, limiter.Allow())
	assert.True(t, limiter.Allow())
	assert.False(t, limiter.Allow())
}

func TestBurstRateLimiter_TokenCap(t *testing.T) {
	ft := &fakeTime{current: time.Unix(0, 0)}
	logger := zaptest.NewLogger(t)
	limiter := NewBurstRateLimiterWithTimeProvider(1, 2, logger, ft.Now)

	// Deplete tokens
	require.True(t, limiter.Allow())
	require.True(t, limiter.Allow())
	assert.False(t, limiter.Allow())

	// Advance a long time (should not exceed burst)
	ft.Advance(10 * time.Second)
	assert.True(t, limiter.Allow())
	assert.True(t, limiter.Allow())
	assert.False(t, limiter.Allow())
}

func TestBurstRateLimiter_Concurrent(t *testing.T) {
	ft := &fakeTime{current: time.Unix(0, 0)}
	logger := zaptest.NewLogger(t)
	limiter := NewBurstRateLimiterWithTimeProvider(5, 5, logger, ft.Now)

	// Simulate concurrent calls
	results := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			results <- limiter.Allow()
		}()
	}
	allowed := 0
	for i := 0; i < 10; i++ {
		if <-results {
			allowed++
		}
	}
	assert.Equal(t, 5, allowed, "should allow up to burst size concurrently")
}
