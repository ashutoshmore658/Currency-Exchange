package schedular

import (
	"context"
	"errors"
	"testing"
	"time"

	"currency-exchange/internals/core/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// --- Mock Cache ---
type mockCache struct {
	setLatestRatesCalls []struct {
		base      domain.Currency
		rates     map[domain.Currency]float64
		timestamp time.Time
	}
}

func (m *mockCache) SetLatestRates(base domain.Currency, rates map[domain.Currency]float64, timestamp time.Time) {
	m.setLatestRatesCalls = append(m.setLatestRatesCalls, struct {
		base      domain.Currency
		rates     map[domain.Currency]float64
		timestamp time.Time
	}{base, rates, timestamp})
}
func (m *mockCache) GetLatestRates(base domain.Currency) (map[domain.Currency]float64, time.Time, bool) {
	return nil, time.Time{}, false
}
func (m *mockCache) SetHistoricalRates(date time.Time, base domain.Currency, rates map[domain.Currency]float64) {
}
func (m *mockCache) GetHistoricalRates(date time.Time, base domain.Currency) (map[domain.Currency]float64, bool) {
	return nil, false
}

// --- Mock API Client ---
type mockAPIClient struct {
	fetchLatestRates func(ctx context.Context, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, time.Time, error)
}

func (m *mockAPIClient) FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, time.Time, error) {
	return m.fetchLatestRates(ctx, base, targets)
}
func (m *mockAPIClient) FetchHistoricalTimeSeriesRates(ctx context.Context, startDate, endDate time.Time, baseCurrency domain.Currency, targetCurrencies []domain.Currency) (*domain.HistoricalTimeSeriesRatesResponse, error) {
	return nil, nil
}

// --- Mock Rate Service ---
type mockRateService struct {
	supportedCurrencies []string
}

func (m *mockRateService) GetSupportedCurrencies() []string                  { return m.supportedCurrencies }
func (m *mockRateService) ValidateCurrencies(currency domain.Currency) error { return nil }
func (m *mockRateService) GetLatestRate(ctx context.Context, base, target domain.Currency) (float64, time.Time, error) {
	return 0, time.Time{}, nil
}
func (m *mockRateService) Convert(ctx context.Context, req domain.ConversionRequest) (*domain.ConversionResult, error) {
	return nil, nil
}
func (m *mockRateService) GetHistoricalRate(ctx context.Context, onDate time.Time, base, target domain.Currency) (float64, error) {
	return 0, nil
}
func (m *mockRateService) GetLatestRates(ctx context.Context, base domain.Currency, targets domain.Currency) (*domain.LatestRates, error) {
	return nil, nil
}
func (m *mockRateService) GetHistoricalRates(ctx context.Context, startDate string, endDate string, base domain.Currency, targets domain.Currency) (*domain.HistoricalRates, error) {
	return nil, nil
}

func TestRefreshCache_AllSuccess(t *testing.T) {
	cache := &mockCache{}
	api := &mockAPIClient{
		fetchLatestRates: func(ctx context.Context, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, time.Time, error) {
			rates := map[domain.Currency]float64{"INR": 82.5}
			return rates, time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC), nil
		},
	}
	rateSvc := &mockRateService{supportedCurrencies: []string{"USD", "INR"}}

	refreshCache(context.Background(), api, cache, rateSvc)

	assert.Equal(t, 2, len(cache.setLatestRatesCalls))
	for _, call := range cache.setLatestRatesCalls {
		assert.Contains(t, []domain.Currency{"USD", "INR"}, call.base)
		assert.Equal(t, 1.0, call.rates[call.base])
		assert.Equal(t, time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC), call.timestamp)
	}
}

func TestRefreshCache_APIError(t *testing.T) {
	cache := &mockCache{}
	api := &mockAPIClient{
		fetchLatestRates: func(ctx context.Context, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, time.Time, error) {
			return nil, time.Time{}, errors.New("api error")
		},
	}
	rateSvc := &mockRateService{supportedCurrencies: []string{"USD", "INR"}}

	refreshCache(context.Background(), api, cache, rateSvc)

	assert.Equal(t, 0, len(cache.setLatestRatesCalls))
}

func TestRefreshCacheWithLockRetry_LockAcquired(t *testing.T) {
	mini, _ := miniredis.Run()
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})

	cache := &mockCache{}
	api := &mockAPIClient{
		fetchLatestRates: func(ctx context.Context, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, time.Time, error) {
			return map[domain.Currency]float64{"INR": 82.5}, time.Now(), nil
		},
	}
	rateSvc := &mockRateService{supportedCurrencies: []string{"USD", "INR"}}

	refreshCacheWithLockRetry(context.Background(), api, cache, redisClient, time.Minute, rateSvc)

	assert.Equal(t, 2, len(cache.setLatestRatesCalls))
}

func TestRefreshCacheWithLockRetry_LockNotAcquired(t *testing.T) {
	mini, _ := miniredis.Run()
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})

	redisClient.Set(context.Background(), "exchange_rate_cache_refresh_lock", "other", time.Minute)

	cache := &mockCache{}
	api := &mockAPIClient{}
	rateSvc := &mockRateService{supportedCurrencies: []string{"USD", "INR"}}

	refreshCacheWithLockRetry(context.Background(), api, cache, redisClient, time.Minute, rateSvc)

	assert.Equal(t, 0, len(cache.setLatestRatesCalls))
}
