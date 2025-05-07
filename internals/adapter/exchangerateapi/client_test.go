package exchangerateapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"currency-exchange/internals/core/domain"

	"github.com/stretchr/testify/assert"
)

// --- Mock FrankFurterAPI ---
type mockFrankFurterAPI struct {
	latestResp *domain.ExchangeResponse
	latestErr  error
	histResp   *domain.HistoricalTimeSeriesRatesResponse
	histErr    error
}

func (m *mockFrankFurterAPI) GetLatest(from string, to []string) (*domain.ExchangeResponse, error) {
	return m.latestResp, m.latestErr
}
func (m *mockFrankFurterAPI) GetHistoricalTimeSeries(from string, to []string, start, end time.Time) (*domain.HistoricalTimeSeriesRatesResponse, error) {
	return m.histResp, m.histErr
}

func TestFetchLatestRates_Success(t *testing.T) {
	mockAPI := &mockFrankFurterAPI{
		latestResp: &domain.ExchangeResponse{
			Base:  "USD",
			Date:  domain.CustomDate(time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC)),
			Rates: map[string]float64{"INR": 82.5, "EUR": 0.9},
		},
	}
	client := NewClient(mockAPI)
	rates, ts, err := client.FetchLatestRates(context.Background(), "USD", []domain.Currency{"INR", "EUR"})
	assert.NoError(t, err)
	assert.Equal(t, 82.5, rates["INR"])
	assert.Equal(t, 0.9, rates["EUR"])
	assert.Equal(t, time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC), ts)
}

func TestFetchLatestRates_Error(t *testing.T) {
	mockAPI := &mockFrankFurterAPI{
		latestErr: errors.New("api down"),
	}
	client := NewClient(mockAPI)
	rates, ts, err := client.FetchLatestRates(context.Background(), "USD", []domain.Currency{"INR"})
	assert.Error(t, err)
	assert.Nil(t, rates)
	assert.True(t, ts.IsZero())
}

func TestFetchHistoricalTimeSeriesRates_Success(t *testing.T) {
	mockAPI := &mockFrankFurterAPI{
		histResp: &domain.HistoricalTimeSeriesRatesResponse{
			Base:      "USD",
			StartDate: "2024-05-01",
			EndDate:   "2024-05-07",
			Rates: map[string]map[string]float64{
				"2024-05-01": {"INR": 80.0},
				"2024-05-07": {"INR": 82.0},
			},
		},
	}
	client := NewClient(mockAPI)
	start := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC)
	resp, err := client.FetchHistoricalTimeSeriesRates(context.Background(), start, end, "USD", []domain.Currency{"INR"})
	assert.NoError(t, err)
	assert.Equal(t, "USD", resp.Base)
	assert.Equal(t, 80.0, resp.Rates["2024-05-01"]["INR"])
	assert.Equal(t, 82.0, resp.Rates["2024-05-07"]["INR"])
}

func TestFetchHistoricalTimeSeriesRates_Error(t *testing.T) {
	mockAPI := &mockFrankFurterAPI{
		histErr: errors.New("api error"),
	}
	client := NewClient(mockAPI)
	start := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC)
	resp, err := client.FetchHistoricalTimeSeriesRates(context.Background(), start, end, "USD", []domain.Currency{"INR"})
	assert.Error(t, err)
	assert.Nil(t, resp)
}
