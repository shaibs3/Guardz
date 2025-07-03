package limiter

import (
	"go.uber.org/zap"
	"sync"
	"time"
)

type TimeProvider func() time.Time

type BurstRateLimiter struct {
	mu           sync.Mutex
	tokens       float64
	lastRefill   time.Time
	limit        float64 // refill rate (tokens per second)
	burst        float64 // max token capacity
	logger       *zap.Logger
	timeProvider TimeProvider
}

func NewBurstRateLimiter(rps int, burst int, logger *zap.Logger) *BurstRateLimiter {
	return NewBurstRateLimiterWithTimeProvider(rps, burst, logger.Named("rate_limiter"), time.Now)
}

func NewBurstRateLimiterWithTimeProvider(rps int, burst int, logger *zap.Logger, tp TimeProvider) *BurstRateLimiter {
	return &BurstRateLimiter{
		limit:        float64(rps),
		burst:        float64(burst),
		tokens:       float64(burst), // start full
		lastRefill:   tp(),
		logger:       logger,
		timeProvider: tp,
	}
}

func (l *BurstRateLimiter) Allow() bool {
	l.mu.Lock()
	defer func() {
		if r := recover(); r != nil {
			l.logger.Error("Panic in rate limiter Allow()", zap.Any("error", r))
		}
		l.mu.Unlock() // always unlock, even if panic
	}()

	now := l.timeProvider()
	elapsed := now.Sub(l.lastRefill).Seconds()

	// refill tokens
	l.tokens += elapsed * l.limit
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
	l.lastRefill = now

	if l.tokens >= 1 {
		l.tokens -= 1
		return true
	}

	l.logger.Warn("Rate limit exceeded", zap.Float64("tokens", l.tokens), zap.Float64("burst", l.burst))
	return false
}
