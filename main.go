package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := connectToRedis("localhost:6379", "", 0)
	flag, err := fixedWindowRateLimiter(context.Background(), rdb, "user:123", 10, 60)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if !flag {
		fmt.Println("Rate limit exceeded at 1")
		return
	}
	flag, err = fixedWindowRateLimiter(context.Background(), rdb, "user:123", 10, 60)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if !flag {
		fmt.Println("Rate limit exceeded at 2")
		return
	}
	flag, err = fixedWindowRateLimiter(context.Background(), rdb, "user:123", 10, 60)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if !flag {
		fmt.Println("Rate limit exceeded at 3")
		return
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
// TODO: Get Key + Count
// TODO: Check in redis
// TODO: Get expiration time and limit as well
// TODO: If key present, increment count otherwise set count to 1
// TODO: If key present and limit < count, throw 429

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
