package repository

import (
	"context"
	"currency-exchange/internals/adapter/exchangerateapi"
	"currency-exchange/internals/core/domain"
	"fmt"
	"log"
	"time"
)

type RateRepository interface {
	GetLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) (rates map[domain.Currency]float64, timestamp time.Time, err error)
	GetHistoricalRates(ctx context.Context, startDate time.Time, endDate time.Time, base domain.Currency, targets []domain.Currency) (*domain.HistoricalTimeSeriesRatesResponse, error)
}

type cachedRateRepository struct {
	apiClient exchangerateapi.RateAPIClient
}

func NewCachedRateRepository(apiClient exchangerateapi.RateAPIClient) RateRepository {
	return &cachedRateRepository{
		apiClient: apiClient,
	}
}

func (r *cachedRateRepository) GetLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, time.Time, error) {
	allSupportedTargets := make([]domain.Currency, 0, len(domain.SupportedCurrencies))
	for curr := range domain.SupportedCurrencies {
		if curr != base {
			allSupportedTargets = append(allSupportedTargets, curr)
		}
	}

	apiRates, apiTimestamp, err := r.apiClient.FetchLatestRates(ctx, base, allSupportedTargets)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to fetch latest rates from API: %w", err)
	}

	fullRates := make(map[domain.Currency]float64)
	for k, v := range apiRates {
		fullRates[k] = v
	}
	fullRates[base] = 1.0

	result := make(map[domain.Currency]float64)
	for _, target := range targets {
		if rate, ok := fullRates[target]; ok {
			result[target] = rate
		} else {
			log.Printf("Warning: API did not return expected rate for target %s (base %s)", target, base)
		}
	}

	return result, apiTimestamp, nil
}

// GetHistoricalRates retrieves historical rates
func (r *cachedRateRepository) GetHistoricalRates(ctx context.Context, startDate time.Time, endDate time.Time, base domain.Currency, targets []domain.Currency) (*domain.HistoricalTimeSeriesRatesResponse, error) {

	apiRates, err := r.apiClient.FetchHistoricalTimeSeriesRates(ctx, startDate, endDate, base, targets)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical rates from API: %w", err)
	}

	return apiRates, nil
}
