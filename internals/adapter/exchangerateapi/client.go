package exchangerateapi

import (
	"context"
	"fmt"
	"log"
	"time"

	"currency-exchange/internals/core/domain"
	"currency-exchange/internals/helpers"
)

// RateAPIClient defines the interface for fetching exchange rates.
type RateAPIClient interface {
	FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, time.Time, error)
	//FetchHistoricalRates(ctx context.Context, date time.Time, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, error)
	FetchHistoricalTimeSeriesRates(ctx context.Context, startDate time.Time, endDate time.Time, baseCurrency domain.Currency, targetCurrencies []domain.Currency) (*domain.HistoricalTimeSeriesRatesResponse, error)
}

type ExRatesClient struct {
	frankFurterAPI helpers.FrankFurterAPI
}

func NewClient(frankFurterAPI helpers.FrankFurterAPI) RateAPIClient {
	return &ExRatesClient{
		frankFurterAPI: frankFurterAPI,
	}
}

func (c *ExRatesClient) FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, time.Time, error) {
	targetStrings := make([]string, len(targets))
	for i, t := range targets {
		targetStrings[i] = string(t)
	}

	log.Printf("Fetching latest rates from API: Base=%s, Targets=%v", base, targetStrings)
	exchangeRates, err := c.frankFurterAPI.GetLatest(string(base), targetStrings)
	if err != nil {
		log.Printf("Error fetching latest rates from API: %v", err)
		return nil, time.Time{}, fmt.Errorf("failed to fetch latest rates from external API: %w", err)
	}

	result := make(map[domain.Currency]float64)
	for currencyStr, rate := range exchangeRates.Rates {
		result[domain.Currency(currencyStr)] = rate
	}

	rateTime := exchangeRates.Date.ToTime()

	log.Printf("Successfully fetched latest rates from API for %s on %s", exchangeRates.Base, exchangeRates.Date.ToTime())
	return result, rateTime, nil
}

// func (c *ExRatesClient) FetchHistoricalRates(ctx context.Context, date time.Time, base domain.Currency, targets []domain.Currency) (map[domain.Currency]float64, error) {
// 	targetStrings := make([]string, len(targets))
// 	for i, t := range targets {
// 		targetStrings[i] = string(t)
// 	}

// 	log.Printf("Fetching historical rates from API: Date=%s, Base=%s, Targets=%v", date.Format("2006-01-02"), base, targetStrings)
// 	rates, err := exrates.On(string(base), date, targetStrings)
// 	if err != nil {
// 		log.Printf("Error fetching historical rates from API: %v", err)
// 		return nil, fmt.Errorf("failed to fetch historical rates from external API: %w", err)
// 	}

// 	result := make(map[domain.Currency]float64)
// 	for currencyStr, rate := range rates.Values {
// 		result[domain.Currency(currencyStr)] = rate
// 	}

// 	log.Printf("Successfully fetched historical rates from API for %s on %s", rates.Base, rates.Date)
// 	return result, nil
// }

func (c *ExRatesClient) FetchHistoricalTimeSeriesRates(ctx context.Context, startDate time.Time, endDate time.Time, baseCurrency domain.Currency, targetCurrencies []domain.Currency) (*domain.HistoricalTimeSeriesRatesResponse, error) {
	targetStrings := make([]string, len(targetCurrencies))
	for i, t := range targetCurrencies {
		targetStrings[i] = string(t)
	}

	log.Printf("Fetching historical rates from API: Date=%s TO Date = %s, Base=%s, Targets=%v", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), baseCurrency, targetStrings)
	rates, err := c.frankFurterAPI.GetHistoricalTimeSeries(string(baseCurrency), targetStrings, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching historical time series rates from API: %v", err)
		return nil, fmt.Errorf("failed to fetch historical timeseries rates from external API: %w", err)
	}

	return rates, nil

}
