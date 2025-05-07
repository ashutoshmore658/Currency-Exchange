package cache

import (
	"context"
	"testing"
	"time"

	"currency-exchange/internals/core/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRedisCache(t *testing.T) *redisCache {
	mini, err := miniredis.Run()
	assert.NoError(t, err)
	client := redis.NewClient(&redis.Options{
		Addr: mini.Addr(),
	})
	return &redisCache{
		client:            client,
		latestRateTTL:     1 * time.Minute,
		historicalRateTTL: 1 * time.Minute,
	}
}

func TestSetAndGetLatestRates_Success(t *testing.T) {
	cache := setupTestRedisCache(t)
	base := domain.Currency("USD")
	rates := map[domain.Currency]float64{"INR": 82.5, "EUR": 0.9}
	timestamp := time.Now().Truncate(time.Second)

	cache.SetLatestRates(base, rates, timestamp)

	gotRates, gotTime, found := cache.GetLatestRates(base)
	assert.True(t, found)
	assert.Equal(t, rates, gotRates)
	assert.WithinDuration(t, timestamp, gotTime, time.Second)
}

func TestGetLatestRates_CacheMiss(t *testing.T) {
	cache := setupTestRedisCache(t)
	gotRates, gotTime, found := cache.GetLatestRates("GBP")
	assert.False(t, found)
	assert.Nil(t, gotRates)
	assert.True(t, gotTime.IsZero())
}

func TestSetAndGetHistoricalRates_Success(t *testing.T) {
	cache := setupTestRedisCache(t)
	date := time.Now().Truncate(24 * time.Hour)
	base := domain.Currency("USD")
	rates := map[domain.Currency]float64{"INR": 80.0, "EUR": 0.91}

	cache.SetHistoricalRates(date, base, rates)

	gotRates, found := cache.GetHistoricalRates(date, base)
	assert.True(t, found)
	assert.Equal(t, rates, gotRates)
}

func TestGetHistoricalRates_CacheMiss(t *testing.T) {
	cache := setupTestRedisCache(t)
	date := time.Now().Truncate(24 * time.Hour)
	gotRates, found := cache.GetHistoricalRates(date, "JPY")
	assert.False(t, found)
	assert.Nil(t, gotRates)
}

func TestGetLatestRates_UnmarshalError(t *testing.T) {
	cache := setupTestRedisCache(t)
	base := domain.Currency("USD")
	key := latestRatesKey(base)

	cache.client.Set(context.Background(), key, "not-json", 1*time.Minute)

	gotRates, gotTime, found := cache.GetLatestRates(base)
	assert.False(t, found)
	assert.Nil(t, gotRates)
	assert.True(t, gotTime.IsZero())
}

func TestGetHistoricalRates_UnmarshalError(t *testing.T) {
	cache := setupTestRedisCache(t)
	date := time.Now().Truncate(24 * time.Hour)
	base := domain.Currency("USD")
	key := historicalRatesKey(date, base)

	cache.client.Set(context.Background(), key, "not-json", 1*time.Minute)

	gotRates, found := cache.GetHistoricalRates(date, base)
	assert.False(t, found)
	assert.Nil(t, gotRates)
}
