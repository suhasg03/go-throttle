package gothrottle

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Leaky Bucket Rate Limitter
func leakyBucketRateLimiter(ctx context.Context, rdb *redis.Client, key string, limit int, windowSeconds int) (bool, error) {
	now := time.Now().Unix()

	luaScript := fmt.Sprintf(`
        local key = KEYS[1]
        local limit = tonumber(ARGV[1])
        local now = tonumber(ARGV[2])
        local window = %d
        
        -- Remove expired timestamps from tail (true "leak")
        while true do
            local oldest = redis.call('LINDEX', key, -1)
            if oldest and (now - tonumber(oldest) > window) then
                redis.call('RPOP', key)
            else
                break
            end
        end
        
        -- Check current count
        local count = redis.call('LLEN', key)
        if count < limit then
            redis.call('LPUSH', key, now)
            redis.call('EXPIRE', key, window)
            return 1
        end
        return 0`, windowSeconds)

	res, err := rdb.Eval(ctx, luaScript, []string{key}, limit, now).Int()
	if err != nil {
		return false, err
	}

	return res == 1, nil
}
