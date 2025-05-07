package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRedis(t *testing.T) *redis.Client {
	mini, err := miniredis.Run()
	assert.NoError(t, err)
	return redis.NewClient(&redis.Options{
		Addr: mini.Addr(),
	})
}

func TestRedisLock_AcquireAndRelease_Success(t *testing.T) {
	client := setupTestRedis(t)
	lock := NewRedisLock(client, "mylock", 2*time.Second)
	ctx := context.Background()

	acquired, err := lock.Acquire(ctx, 2*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired)

	err = lock.Release(ctx)
	assert.NoError(t, err)
}

func TestRedisLock_Acquire_Timeout(t *testing.T) {
	client := setupTestRedis(t)
	ctx := context.Background()

	lock1 := NewRedisLock(client, "mylock", 5*time.Second)
	acquired, err := lock1.Acquire(ctx, 1*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired)

	lock2 := NewRedisLock(client, "mylock", 5*time.Second)
	start := time.Now()
	acquired2, err := lock2.Acquire(ctx, 500*time.Millisecond)
	elapsed := time.Since(start)
	assert.Error(t, err)
	assert.False(t, acquired2)
	assert.GreaterOrEqual(t, elapsed, 500*time.Millisecond)

	_ = lock1.Release(ctx)
}

func TestRedisLock_Release_NotOwner(t *testing.T) {
	client := setupTestRedis(t)
	ctx := context.Background()

	lock1 := NewRedisLock(client, "mylock", 5*time.Second)
	acquired, err := lock1.Acquire(ctx, 1*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired)

	lock2 := NewRedisLock(client, "mylock", 5*time.Second)
	err = lock2.Release(ctx)
	assert.NoError(t, err)

	acquiredAgain, err := lock1.Acquire(ctx, 500*time.Millisecond)
	assert.Error(t, err)
	assert.False(t, acquiredAgain)

	_ = lock1.Release(ctx)
}

func TestRedisLock_Acquire_ReacquireAfterTTL(t *testing.T) {
	mini, err := miniredis.Run()
	assert.NoError(t, err)
	client := redis.NewClient(&redis.Options{
		Addr: mini.Addr(),
	})
	ctx := context.Background()

	lock1 := NewRedisLock(client, "mylock", 500*time.Millisecond)
	acquired, err := lock1.Acquire(ctx, 1*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired)

	mini.FastForward(600 * time.Millisecond)

	lock2 := NewRedisLock(client, "mylock", 1*time.Second)
	acquired2, err := lock2.Acquire(ctx, 1*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired2)

	_ = lock2.Release(ctx)
}
