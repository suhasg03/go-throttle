package gothrottle

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type GoThrottle struct {
	Redis    *redis.Client
	Strategy RateLimitStrategy
}

type RateLimitStrategy int

const (
	FixedWindow RateLimitStrategy = iota
	SlidingWindow
	LeakyBucket
)

func NewGoThrottle(rdb *redis.Client, strategy RateLimitStrategy) *GoThrottle {
	return &GoThrottle{
		Redis:    rdb,
		Strategy: strategy,
	}
}

func (gt *GoThrottle) Allow(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	switch gt.Strategy {
	case FixedWindow:
		return fixedWindowRateLimiter(ctx, gt.Redis, key, limit, windowSeconds)
	case SlidingWindow:
		return slidingWindowRateLimiter(ctx, gt.Redis, key, limit, windowSeconds)
	case LeakyBucket:
		return leakyBucketRateLimiter(ctx, gt.Redis, key, limit, windowSeconds)
	default:
		return false, fmt.Errorf("unknown rate limiting strategy")
	}
}
