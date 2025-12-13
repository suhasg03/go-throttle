package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := connectToRedis("localhost:6379", "", 0)
	for {
		flag, err := slidingWindowRateLimiter(context.Background(), rdb, "user:123", 10, 60)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		if !flag {
			fmt.Println("Rate limit exceeded at 1")
			return
		}
		flag, err = slidingWindowRateLimiter(context.Background(), rdb, "user:123", 10, 60)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		if !flag {
			fmt.Println("Rate limit exceeded at 2")
			return
		}
		flag, err = slidingWindowRateLimiter(context.Background(), rdb, "user:123", 10, 60)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		if !flag {
			fmt.Println("Rate limit exceeded at 3")
			return
		}
	}
}

func connectToRedis(Addr string, Password string, DB int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     Addr,
		Password: Password,
		DB:       DB,
	})
}

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

// Leaky Bucket Rate Limitter
func trueLeakyBucket(ctx context.Context, rdb *redis.Client, key string, limit int, windowSeconds int) (bool, error) {
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
