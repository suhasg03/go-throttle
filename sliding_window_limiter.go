package gothrottle

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Sliding Window Rate Limitter
func slidingWindowRateLimiter(ctx context.Context, rdb *redis.Client, key string, limit int, windowSeconds int) (bool, error) {
	now := time.Now().Unix()

	luaScript := fmt.Sprintf(`
        local key = KEYS[1]
        local limit = tonumber(ARGV[1])
        local now = tonumber(ARGV[2])
        local window = %d
        
        -- Remove timestamps outside sliding window (now - window)
        while true do
            local oldest = redis.call('LINDEX', key, -1)
            if oldest and (now - tonumber(oldest) > window) then
                redis.call('RPOP', key)
            else
                break
            end
        end
        
        -- Count current valid requests in window
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
