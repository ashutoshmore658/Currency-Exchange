package repository

import (
	"context"
	"currency-exchange/internals/adapter/cache"
	"currency-exchange/internals/adapter/exchangerateapi"
	"currency-exchange/internals/core/domain"
	"fmt"
	"log"
	"time"
)

type RateRepository interface {
	GetLatestRates(ctx context.Context, base domain.Currency, targets domain.Currency) (rates map[domain.Currency]float64, timestamp time.Time, err error)
	GetHistoricalRates(ctx context.Context, startDate time.Time, endDate time.Time, base domain.Currency, targets domain.Currency) (map[time.Time]float64, error)
}

type cachedRateRepository struct {
	apiClient exchangerateapi.RateAPIClient
	cache     cache.Cache
}

func NewCachedRateRepository(apiClient exchangerateapi.RateAPIClient, cache cache.Cache) RateRepository {
	return &cachedRateRepository{
		apiClient: apiClient,
		cache:     cache,
	}
}

func (r *cachedRateRepository) GetLatestRates(ctx context.Context, base domain.Currency, target domain.Currency) (map[domain.Currency]float64, time.Time, error) {
	cachedRates, timestamp, found := r.cache.GetLatestRates(base)
	if found {
		result := make(map[domain.Currency]float64)
		if rate, ok := cachedRates[target]; ok {
			result[target] = rate
		}

		result[base] = 1.0
		return result, timestamp, nil
	}

	allSupportedTargets := make([]domain.Currency, 0, len(domain.SupportedCurrencies))
	for curr := range domain.SupportedCurrencies {
		if curr != base { // API doesn't return base=base
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
	fullRates[base] = 1.0 // Rate of base to itself is always 1

	go r.cache.SetLatestRates(base, fullRates, apiTimestamp)

	result := make(map[domain.Currency]float64)
	if rate, ok := fullRates[target]; ok {
		result[target] = rate
	} else {
		log.Printf("Warning: API did not return expected rate for target %s (base %s)", target, base)
	}
	result[base] = 1.0

	return result, apiTimestamp, nil
}

// GetHistoricalRates retrieves historical rates
func (r *cachedRateRepository) GetHistoricalRates(ctx context.Context, startDate time.Time, endDate time.Time, base domain.Currency, target domain.Currency) (map[time.Time]float64, error) {
	resultantDateToRateMap := make(map[time.Time]float64)
	allFound := true
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		cachedRates, found := r.cache.GetHistoricalRates(date, base)
		if found {
			rate, ok := cachedRates[target]
			if !ok {
				log.Printf("Did not recieive anything in cache map for target currency : %v", target)
			}
			resultantDateToRateMap[date] = rate
		} else {
			allFound = false
			break
		}

	}
	if allFound {
		return resultantDateToRateMap, nil
	}

	allSupportedTargets := make([]domain.Currency, 0, len(domain.SupportedCurrencies))
	for curr := range domain.SupportedCurrencies {
		if curr != base {
			allSupportedTargets = append(allSupportedTargets, curr)
		}
	}

	apiRates, err := r.apiClient.FetchHistoricalTimeSeriesRates(ctx, startDate, endDate, base, allSupportedTargets)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical rates from API: %w", err)
	}
	cacheCurrencyMap := make(map[domain.Currency]float64)
	rates := apiRates.Rates
	for date, currencyRateMap := range rates {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			log.Printf("An Error occurred while parsing the string date so not adding it to resultant map\n")
			continue
		}
		for currency, rate := range currencyRateMap {
			if currency == string(target) {
				resultantDateToRateMap[parsedDate] = rate
			}
			cacheCurrencyMap[domain.Currency(currency)] = rate
		}

		go r.cache.SetHistoricalRates(parsedDate, base, cacheCurrencyMap)

	}

	return resultantDateToRateMap, nil
}
