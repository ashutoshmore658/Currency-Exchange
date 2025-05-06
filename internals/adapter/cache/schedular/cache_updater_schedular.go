package schedular

import (
	"context"
	"currency-exchange/internals/adapter/cache"
	"currency-exchange/internals/adapter/exchangerateapi"
	"currency-exchange/internals/core/domain"
	"currency-exchange/internals/service"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func StartBackgroundRefreshWithLock(ctx context.Context, interval time.Duration, apiClient exchangerateapi.RateAPIClient, cache cache.Cache, redisClient *redis.Client, rateService service.RateService) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Background refresh worker started. Refresh interval: %s", interval)

	refreshCacheWithLockRetry(ctx, apiClient, cache, redisClient, interval, rateService)

	for {
		select {
		case <-ticker.C:
			log.Println("Background refresh triggered.")
			refreshCacheWithLockRetry(ctx, apiClient, cache, redisClient, interval, rateService)
		case <-ctx.Done():
			log.Println("Background refresh worker stopping.")
			return
		}
	}
}

func refreshCacheWithLockRetry(ctx context.Context, apiClient exchangerateapi.RateAPIClient, cacheObject cache.Cache, redisClient *redis.Client, interval time.Duration, rateService service.RateService) {
	const lockKey = "exchange_rate_cache_refresh_lock"
	lockTTL := 2 * time.Minute
	maxWait := 15 * time.Second

	lock := cache.NewRedisLock(redisClient, lockKey, lockTTL)
	acquired, err := lock.Acquire(ctx, maxWait)
	if err != nil {
		log.Printf("Error acquiring distributed lock for cache refresh: %v", err)
		return
	}
	if !acquired {
		log.Println("Could not acquire lock for cache refresh after waiting, skipping this cycle")
		return
	}
	defer func() {
		if err := lock.Release(context.Background()); err != nil {
			log.Printf("Error releasing distributed lock: %v", err)
		}
	}()

	refreshCache(ctx, apiClient, cacheObject, rateService)
}

func refreshCache(ctx context.Context, client exchangerateapi.RateAPIClient, cache cache.Cache, rateService service.RateService) {
	allCurrencies := rateService.GetSupportedCurrencies()
	for _, base := range allCurrencies {
		targets := make([]domain.Currency, 0, len(allCurrencies)-1)
		for _, target := range allCurrencies {
			if target != base {
				targets = append(targets, domain.Currency(target))
			}
		}
		if len(targets) == 0 {
			continue
		}

		rates, timestamp, err := client.FetchLatestRates(ctx, domain.Currency(base), targets)
		if err != nil {
			log.Printf("ERROR refreshing cache for base %s: %v", base, err)
			continue
		}

		rates[domain.Currency(base)] = 1.0
		cache.SetLatestRates(domain.Currency(base), rates, timestamp)
		log.Printf("Cache refreshed successfully for base %s", base)
	}
}
