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

const (
	baseURL = "https://api.frankfurter.app/"
	dateFmt = "2006-01-02"
)

func GetLatest(fromCurrency string, toCurrencies []string) (*domain.ExchangeResponse, error) {
	log.Printf("Fetching latest currecy exchange rates using %v API, for base %v urrency to target currecies %v", baseURL, fromCurrency, toCurrencies)
	response := &domain.ExchangeResponse{}
	err := doRequest(baseURL+"latest", makeParams(fromCurrency, toCurrencies), response)
	if err != nil {
		return nil, err
	}

	return response, nil

}

func GetHistoricalTimeSeries(fromCurrency string, toCurrency []string, startDate time.Time, endDate time.Time) (*domain.HistoricalTimeSeriesRatesResponse, error) {
	log.Printf("Fetching historical currecy exchange rates using %v API, for base %v urrency to target currecies %v from day %v to day %v", baseURL, fromCurrency, toCurrency, startDate, endDate)
	response := &domain.HistoricalTimeSeriesRatesResponse{}
	err := doRequest(baseURL+startDate.Format(dateFmt)+".."+endDate.Format(dateFmt), makeParams(fromCurrency, toCurrency), response)

	if err != nil {
		return nil, err
	}

	return response, nil

}

func doRequest(url string, params url.Values, w interface{}) error {
	if len(params) > 0 {
		url = fmt.Sprintf("%s?%s", url, params.Encode())
	}

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(w)
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
