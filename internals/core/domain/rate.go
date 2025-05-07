package domain

import (
	"strings"
	"time"
)

// Currency represents a currency code (e.g., "USD", "INR").
type Currency string

// SupportedCurrencies lists the currencies the service handles.
var SupportedCurrencies = map[Currency]bool{
	"USD": true,
	"INR": true,
	"EUR": true,
	"JPY": true,
	"GBP": true,
}

// IsSupported checks if a currency code is supported.
func (c Currency) IsSupported() bool {
	_, ok := SupportedCurrencies[c]
	return ok
}

type CustomDate time.Time

func (cd *CustomDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	*cd = CustomDate(t)
	return nil
}

func (cd CustomDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(cd).Format("2006-01-02") + `"`), nil
}

func (cd CustomDate) ToTime() time.Time {
	return time.Time(cd)
}

type ExchangeResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   CustomDate         `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

type LatestRates struct {
	Base      Currency             `json:"base"`
	Rates     map[Currency]float64 `json:"rates"`
	Timestamp int64                `json:"timestamp"` // Unix timestamp
}

type HistoricalRates struct {
	Base   Currency              `json:"base"`
	Rates  map[time.Time]float64 `json:"rates"`
	Amount float64               `json:"amount"`
	Target Currency              `json:"target"`
}

type HistoricalTimeSeriesRatesResponse struct {
	Amount    float64                       `json:"amount"`
	Base      string                        `json:"base"`
	StartDate string                        `json:"start_date"`
	EndDate   string                        `json:"end_date"`
	Rates     map[string]map[string]float64 `json:"rates"`
}

type ConversionRequest struct {
	From   Currency   `json:"from"`
	To     Currency   `json:"to"`
	Amount float64    `json:"amount"`
	Date   *time.Time `json:"date,omitempty"`
}

type ConversionResult struct {
	From            Currency   `json:"from"`
	To              Currency   `json:"to"`
	OriginalAmount  float64    `json:"amount"`
	ConvertedAmount float64    `json:"convertedAmount"`
	Rate            float64    `json:"rate"`
	Date            *time.Time `json:"onDate,omitempty"`
}
