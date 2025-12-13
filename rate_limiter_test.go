package gothrottle

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use test DB
	})

	// Test connection
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis not available:", err)
	}

	// Flush test DB before each test
	require.NoError(t, rdb.FlushDB(ctx).Err())

	t.Cleanup(func() {
		rdb.FlushDB(ctx)
		rdb.Close()
	})

	return rdb
}

func TestNewGoThrottle(t *testing.T) {
	rdb := setupTestRedis(t)
	defer rdb.Close()

	throttle := NewGoThrottle(rdb, FixedWindow)
	assert.Equal(t, FixedWindow, throttle.Strategy)
	assert.Equal(t, rdb, throttle.Redis)
}

func TestGoThrottle_Allow_FixedWindow(t *testing.T) {
	rdb := setupTestRedis(t)
	throttle := NewGoThrottle(rdb, FixedWindow)
	ctx := context.Background()
	key := "test:user:1"
	limit := 3
	window := 1

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		allowed, err := throttle.Allow(ctx, key, limit, window)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// 4th request should be rejected
	allowed, err := throttle.Allow(ctx, key, limit, window)
	assert.NoError(t, err)
	assert.False(t, allowed)

	// Wait for window reset and test again
	time.Sleep(2 * time.Second)
	allowed, err = throttle.Allow(ctx, key, limit, window)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestGoThrottle_Allow_SlidingWindow(t *testing.T) {
	rdb := setupTestRedis(t)
	throttle := NewGoThrottle(rdb, SlidingWindow)
	ctx := context.Background()
	key := "test:user:2"
	limit := 3
	window := 2

	// Burst test: exactly 3 allowed
	for i := 0; i < 3; i++ {
		allowed, err := throttle.Allow(ctx, key, limit, window)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	// 4th should be rejected immediately
	allowed, err := throttle.Allow(ctx, key, limit, window)
	assert.NoError(t, err)
	assert.False(t, allowed)

	// Wait longer than window, should allow again
	time.Sleep(3 * time.Second)
	allowed, err = throttle.Allow(ctx, key, limit, window)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestGoThrottle_Allow_LeakyBucket(t *testing.T) {
	rdb := setupTestRedis(t)
	throttle := NewGoThrottle(rdb, LeakyBucket)
	ctx := context.Background()
	key := "test:user:3"
	limit := 3
	window := 2

	// Same behavior as sliding window in this implementation
	for i := 0; i < 3; i++ {
		allowed, err := throttle.Allow(ctx, key, limit, window)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	allowed, err := throttle.Allow(ctx, key, limit, window)
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestGoThrottle_Allow_UnknownStrategy(t *testing.T) {
	rdb := setupTestRedis(t)
	throttle := NewGoThrottle(rdb, RateLimitStrategy(99)) // Invalid strategy
	ctx := context.Background()

	allowed, err := throttle.Allow(ctx, "test", 10, 60)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown rate limiting strategy")
	assert.False(t, allowed)
}

func TestGoThrottle_MultipleKeys(t *testing.T) {
	rdb := setupTestRedis(t)
	throttle := NewGoThrottle(rdb, FixedWindow)
	ctx := context.Background()

	// Different keys should not interfere
	key1 := "user:alice"
	key2 := "user:bob"
	limit := 2

	// Alice makes 3 requests (3rd rejected)
	for i := 0; i < 3; i++ {
		allowed, err := throttle.Allow(ctx, key1, limit, 1)
		assert.NoError(t, err)
		if i < 2 {
			assert.True(t, allowed)
		} else {
			assert.False(t, allowed)
		}
	}

	// Bob should still get full quota
	for i := 0; i < 2; i++ {
		allowed, err := throttle.Allow(ctx, key2, limit, 1)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}
}

func TestGoThrottle_ZeroLimit(t *testing.T) {
	rdb := setupTestRedis(t)
	throttle := NewGoThrottle(rdb, FixedWindow)
	ctx := context.Background()

	allowed, err := throttle.Allow(ctx, "test:user:zero", 0, 60)
	assert.NoError(t, err)
	assert.False(t, allowed)
}
