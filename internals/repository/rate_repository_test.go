package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"currency-exchange/internals/core/domain"

	"github.com/stretchr/testify/assert"
)

// --- Mock Cache ---
type mockCache struct {
	latestRates     map[domain.Currency]float64
	latestTimestamp time.Time
	latestFound     bool
	histRates       map[domain.Currency]float64
	histFound       bool
	setHistCalled   chan struct{}
	setLatestCalled chan struct{}
}

func (m *mockCache) SetLatestRates(base domain.Currency, rates map[domain.Currency]float64, timestamp time.Time) {
	if m.setLatestCalled != nil {
		m.setLatestCalled <- struct{}{}
	}
	m.latestRates = rates
	m.latestTimestamp = timestamp
}

func (m *mockCache) GetLatestRates(base domain.Currency) (map[domain.Currency]float64, time.Time, bool) {
	return m.latestRates, m.latestTimestamp, m.latestFound
}

func (m *mockCache) SetHistoricalRates(date time.Time, base domain.Currency, rates map[domain.Currency]float64) {
	if m.setHistCalled != nil {
		m.setHistCalled <- struct{}{}
	}
	m.histRates = rates
}

func (m *mockCache) GetHistoricalRates(date time.Time, base domain.Currency) (map[domain.Currency]float64, bool) {
	return m.histRates, m.histFound
}

// --- Mock API Client ---
type mockAPIClient struct {
	latestRatesResp    map[domain.Currency]float64
	latestRatesTime    time.Time
	latestRatesErr     error
	histTimeSeriesResp *domain.HistoricalTimeSeriesRatesResponse
	histTimeSeriesErr  error
}

func (m *mockAPIClient) FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, time.Time, error) {
	return m.latestRatesResp, m.latestRatesTime, m.latestRatesErr
}

func (m *mockAPIClient) FetchHistoricalTimeSeriesRates(ctx context.Context, startDate, endDate time.Time, baseCurrency domain.Currency, targetCurrencies []domain.Currency) (*domain.HistoricalTimeSeriesRatesResponse, error) {
	return m.histTimeSeriesResp, m.histTimeSeriesErr
}

func TestGetLatestRates_CacheHit(t *testing.T) {
	cache := &mockCache{
		latestRates:     map[domain.Currency]float64{"INR": 82.5},
		latestTimestamp: time.Now(),
		latestFound:     true,
	}
	repo := NewCachedRateRepository(nil, cache)
	rates, ts, err := repo.GetLatestRates(context.Background(), "USD", "INR")
	assert.NoError(t, err)
	assert.Equal(t, 82.5, rates["INR"])
	assert.Equal(t, 1.0, rates["USD"])
	assert.WithinDuration(t, time.Now(), ts, time.Second)
}

func TestGetLatestRates_CacheMiss_APISuccess(t *testing.T) {
	ch := make(chan struct{}, 1)
	cache := &mockCache{latestFound: false, setLatestCalled: ch}
	api := &mockAPIClient{
		latestRatesResp: map[domain.Currency]float64{"INR": 82.5, "EUR": 0.9},
		latestRatesTime: time.Now(),
	}
	repo := NewCachedRateRepository(api, cache)
	rates, ts, err := repo.GetLatestRates(context.Background(), "USD", "INR")
	assert.NoError(t, err)
	assert.Equal(t, 82.5, rates["INR"])
	assert.Equal(t, 1.0, rates["USD"])
	assert.WithinDuration(t, time.Now(), ts, time.Second)
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Error("SetLatestRates was not called in time")
	}
}

func TestGetLatestRates_CacheMiss_APINoTarget(t *testing.T) {
	cache := &mockCache{latestFound: false}
	api := &mockAPIClient{
		latestRatesResp: map[domain.Currency]float64{"EUR": 0.9},
		latestRatesTime: time.Now(),
	}
	repo := NewCachedRateRepository(api, cache)
	rates, ts, err := repo.GetLatestRates(context.Background(), "USD", "INR")
	assert.NoError(t, err)
	assert.NotContains(t, rates, "INR")
	assert.Equal(t, 1.0, rates["USD"])
	assert.WithinDuration(t, time.Now(), ts, time.Second)
}

func TestGetLatestRates_APIFails(t *testing.T) {
	cache := &mockCache{latestFound: false}
	api := &mockAPIClient{
		latestRatesErr: errors.New("api error"),
	}
	repo := NewCachedRateRepository(api, cache)
	rates, ts, err := repo.GetLatestRates(context.Background(), "USD", "INR")
	assert.Error(t, err)
	assert.Nil(t, rates)
	assert.True(t, ts.IsZero())
}

func TestGetHistoricalRates_AllCacheHit(t *testing.T) {
	date := time.Now().Truncate(24 * time.Hour)
	cache := &mockCache{
		histRates: map[domain.Currency]float64{"INR": 80.0},
		histFound: true,
	}
	repo := NewCachedRateRepository(nil, cache)
	rates, err := repo.GetHistoricalRates(context.Background(), date, date, "USD", "INR")
	assert.NoError(t, err)
	assert.Equal(t, 80.0, rates[date])
}

func TestGetHistoricalRates_CacheMiss_APISuccess(t *testing.T) {
	date := time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC)
	ch := make(chan struct{}, 1)
	cache := &mockCache{
		histRates:     map[domain.Currency]float64{"INR": 0},
		histFound:     false,
		setHistCalled: ch,
	}
	api := &mockAPIClient{
		histTimeSeriesResp: &domain.HistoricalTimeSeriesRatesResponse{
			Rates: map[string]map[string]float64{
				"2024-05-07": {"INR": 81.0, "EUR": 0.9},
			},
		},
	}
	repo := NewCachedRateRepository(api, cache)
	rates, err := repo.GetHistoricalRates(context.Background(), date, date, "USD", "INR")
	assert.NoError(t, err)
	assert.Equal(t, 81.0, rates[date])
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Error("SetHistoricalRates was not called in time")
	}
}

func TestGetHistoricalRates_CacheMiss_APIFails(t *testing.T) {
	date := time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC)
	cache := &mockCache{
		histRates: map[domain.Currency]float64{"INR": 0},
		histFound: false,
	}
	api := &mockAPIClient{
		histTimeSeriesErr: errors.New("api error"),
	}
	repo := NewCachedRateRepository(api, cache)
	rates, err := repo.GetHistoricalRates(context.Background(), date, date, "USD", "INR")
	assert.Error(t, err)
	assert.Nil(t, rates)
}

func TestGetHistoricalRates_APIReturnsBadDate(t *testing.T) {
	date := time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC)
	cache := &mockCache{
		histRates: map[domain.Currency]float64{"INR": 0},
		histFound: false,
	}
	api := &mockAPIClient{
		histTimeSeriesResp: &domain.HistoricalTimeSeriesRatesResponse{
			Rates: map[string]map[string]float64{
				"bad-date": {"INR": 81.0},
			},
		},
	}
	repo := NewCachedRateRepository(api, cache)
	rates, err := repo.GetHistoricalRates(context.Background(), date, date, "USD", "INR")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(rates))
}
