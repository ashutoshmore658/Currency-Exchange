package cache

import (
	"context"
	"currency-exchange/internals/core/domain"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	SetLatestRates(base domain.Currency, rates map[domain.Currency]float64, timestamp time.Time)
	GetLatestRates(base domain.Currency) (map[domain.Currency]float64, time.Time, bool)
	SetHistoricalRates(date time.Time, base domain.Currency, rates map[domain.Currency]float64)
	GetHistoricalRates(date time.Time, base domain.Currency) (map[domain.Currency]float64, bool)
}

type redisCache struct {
	client            *redis.Client
	latestRateTTL     time.Duration
	historicalRateTTL time.Duration
}

func NewRedisCache(client *redis.Client, latestTTL, historicalTTL time.Duration) Cache {
	return &redisCache{
		client:            client,
		latestRateTTL:     latestTTL,
		historicalRateTTL: historicalTTL,
	}
}

func latestRatesKey(base domain.Currency) string {
	return fmt.Sprintf("latest:%s", base)
}

func historicalRatesKey(date time.Time, base domain.Currency) string {
	return fmt.Sprintf("historical:%s:%s", date.Format("2006-01-02"), base)
}

type cachedLatestRatesData struct {
	Rates     map[domain.Currency]float64 `json:"rates"`
	Timestamp time.Time                   `json:"timestamp"`
}

func (rc *redisCache) SetLatestRates(base domain.Currency, rates map[domain.Currency]float64, timestamp time.Time) {
	lock := NewRedisLock(rc.client, "cache_write_lock", 30*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // max wait 10s to acquire lock
	defer cancel()

	acquired, err := lock.Acquire(ctx, 10*time.Second)
	if err != nil {
		log.Printf("Error acquiring lock for SetLatestRates: %v", err)
		return
	}
	if !acquired {
		log.Println("Could not acquire lock for SetLatestRates after waiting")
		return
	}
	defer func() {
		if err := lock.Release(context.Background()); err != nil {
			log.Printf("Error releasing lock for SetLatestRates: %v", err)
		}
	}()

	key := latestRatesKey(base)
	data := cachedLatestRatesData{
		Rates:     rates,
		Timestamp: timestamp,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling latest rates: %v", err)
		return
	}

	err = rc.client.Set(ctx, key, jsonData, rc.latestRateTTL).Err()
	if err != nil {
		log.Printf("Error setting latest rates in Redis: %v", err)
	} else {
		log.Printf("Cached latest rates for %s in Redis with TTL %s", base, rc.latestRateTTL)
	}
}

func (rc *redisCache) GetLatestRates(base domain.Currency) (map[domain.Currency]float64, time.Time, bool) {
	key := latestRatesKey(base)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jsonData, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			log.Printf("Cache miss for key %s", key)
			return nil, time.Time{}, false
		}
		log.Printf("Error getting latest rates from Redis: %v", err)
		return nil, time.Time{}, false
	}

	var data cachedLatestRatesData
	err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		log.Printf("Error unmarshaling latest rates JSON: %v", err)
		return nil, time.Time{}, false
	}

	log.Printf("Cache hit for key %s", key)
	return data.Rates, data.Timestamp, true
}

func (rc *redisCache) SetHistoricalRates(date time.Time, base domain.Currency, rates map[domain.Currency]float64) {
	lock := NewRedisLock(rc.client, "cache_write_lock", 30*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // max wait 10s to acquire lock
	defer cancel()

	acquired, err := lock.Acquire(ctx, 10*time.Second)
	if err != nil {
		log.Printf("Error acquiring lock for SetHistoricalRates: %v", err)
		return
	}
	if !acquired {
		log.Println("Could not acquire lock for SetHistoricalRates after waiting")
		return
	}
	defer func() {
		if err := lock.Release(context.Background()); err != nil {
			log.Printf("Error releasing lock for SetHistoricalRates: %v", err)
		}
	}()

	key := historicalRatesKey(date, base)

	jsonData, err := json.Marshal(rates)
	if err != nil {
		log.Printf("Error marshaling historical rates: %v", err)
		return
	}

	err = rc.client.Set(ctx, key, jsonData, rc.historicalRateTTL).Err()
	if err != nil {
		log.Printf("Error setting historical rates in Redis: %v", err)
	} else {
		log.Printf("Cached historical rates for %s %s in Redis with TTL %s", base, date.Format("2006-01-02"), rc.historicalRateTTL)
	}
}

func (rc *redisCache) GetHistoricalRates(date time.Time, base domain.Currency) (map[domain.Currency]float64, bool) {
	key := historicalRatesKey(date, base)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jsonData, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			log.Printf("Cache miss for key %s", key)
			return nil, false
		}
		log.Printf("Error getting historical rates from Redis: %v", err)
		return nil, false
	}

	var rates map[domain.Currency]float64
	err = json.Unmarshal([]byte(jsonData), &rates)
	if err != nil {
		log.Printf("Error unmarshaling historical rates JSON: %v", err)
		return nil, false
	}

	log.Printf("Cache hit for key %s", key)
	return rates, true
}
