package cache

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisLock struct {
	client *redis.Client
	key    string
	value  string
	ttl    time.Duration
}

// NewRedisLock creates a new lock with a unique value and TTL
func NewRedisLock(client *redis.Client, key string, ttl time.Duration) *RedisLock {
	return &RedisLock{
		client: client,
		key:    key,
		value:  uuid.NewString(),
		ttl:    ttl,
	}
}

// Acquire tries to acquire the lock with retries and backoff.
// It waits up to maxWait duration trying to get the lock.
func (l *RedisLock) Acquire(ctx context.Context, maxWait time.Duration) (bool, error) {
	deadline := time.Now().Add(maxWait)
	for {
		ok, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
		if err != nil {
			return false, err
		}
		if ok {
			// Lock acquired
			return true, nil
		}

		// Lock not acquired, check if deadline exceeded
		if time.Now().After(deadline) {
			return false, errors.New("timeout acquiring redis lock")
		}

		// Wait a bit before retrying
		time.Sleep(100 * time.Millisecond)
	}
}

// Release releases the lock only if owned by this instance
func (l *RedisLock) Release(ctx context.Context) error {
	luaScript := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`
	res, err := l.client.Eval(ctx, luaScript, []string{l.key}, l.value).Result()
	if err != nil {
		return err
	}
	if res.(int64) == 0 {
		log.Println("Lock not released: it was owned by someone else or expired")
	}
	return nil
}
