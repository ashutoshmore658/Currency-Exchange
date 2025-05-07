package helpers

import (
	"currency-exchange/internals/core/domain"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// const (
// 	baseURL = "https://api.frankfurter.app/"
// 	dateFmt = "2006-01-02"
// )

type FrankFurterAPI interface {
	GetLatest(fromCurrency string, toCurrencies []string) (*domain.ExchangeResponse, error)
	GetHistoricalTimeSeries(fromCurrency string, toCurrency []string, startDate time.Time, endDate time.Time) (*domain.HistoricalTimeSeriesRatesResponse, error)
}

type FrankFurterAPIClient struct {
	baseURL string
	dateFmt string
}

func NewFrankFurterAPI(baseURL, dateFmt string) FrankFurterAPI {
	return &FrankFurterAPIClient{
		baseURL: baseURL,
		dateFmt: dateFmt,
	}
}

func (f *FrankFurterAPIClient) GetLatest(fromCurrency string, toCurrencies []string) (*domain.ExchangeResponse, error) {
	log.Printf("Fetching latest currecy exchange rates using %v API, for base %v urrency to target currecies %v", f.baseURL, fromCurrency, toCurrencies)
	response := &domain.ExchangeResponse{}
	err := doRequest(f.baseURL+"latest", makeParams(fromCurrency, toCurrencies), response)
	if err != nil {
		return nil, err
	}

	return response, nil

}

func (f *FrankFurterAPIClient) GetHistoricalTimeSeries(fromCurrency string, toCurrency []string, startDate time.Time, endDate time.Time) (*domain.HistoricalTimeSeriesRatesResponse, error) {
	log.Printf("Fetching historical currecy exchange rates using %v API, for base %v urrency to target currecies %v from day %v to day %v", f.baseURL, fromCurrency, toCurrency, startDate, endDate)
	response := &domain.HistoricalTimeSeriesRatesResponse{}
	err := doRequest(f.baseURL+startDate.Format(f.dateFmt)+".."+endDate.Format(f.dateFmt), makeParams(fromCurrency, toCurrency), response)

	if err != nil {
		return nil, err
	}

	return response, nil

}

// func doRequest(url string, params url.Values, w interface{}) error {
// 	if len(params) > 0 {
// 		url = fmt.Sprintf("%s?%s", url, params.Encode())
// 	}

// 	client := &http.Client{
// 		Timeout: time.Second * 30,
// 	}

// 	resp, err := client.Get(url)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	return json.NewDecoder(resp.Body).Decode(w)
// }

func doRequest(url string, params url.Values, w interface{}) error {
	if len(params) > 0 {
		url = fmt.Sprintf("%s?%s", url, params.Encode())
	}

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	var lastErr error
	baseDelay := time.Second
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(url)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return json.NewDecoder(resp.Body).Decode(w)
			}
			// Treat non-200 as error
			lastErr = fmt.Errorf("http status %d", resp.StatusCode)
			return lastErr
		}
		// Network error, retry
		lastErr = err
		time.Sleep(baseDelay * (1 << i))
	}
	return fmt.Errorf("external API error after %d retries: %w", maxRetries, lastErr)
}

func makeParams(base string, currencies []string) url.Values {
	params := url.Values{}
	if base := strings.ToUpper(strings.TrimSpace(base)); base != "" {
		params.Add("from", base)
	}

	symbols := []string{}
	for _, currency := range currencies {
		symbol := strings.ToUpper(strings.TrimSpace(currency))
		if symbol != "" {
			symbols = append(symbols, symbol)
		}
	}

	if len(symbols) > 0 {
		params.Add("to", strings.Join(symbols, ","))
	}

	return params
}
