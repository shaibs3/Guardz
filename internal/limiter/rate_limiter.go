package limiter

type RateLimiter interface {
	Allow() bool
}
