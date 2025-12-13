package gothrottle

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Fixed Window Rate Limitter
func fixedWindowRateLimiter(ctx context.Context, rdb *redis.Client, key string, limit int, windowSeconds int) (bool, error) {
	window := time.Duration(windowSeconds) * time.Second

	pipe := rdb.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count, err := cmds[0].(*redis.IntCmd).Result()
	if err != nil {
		return false, err
	}

	return count <= int64(limit), nil
}
