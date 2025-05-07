package helpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"currency-exchange/internals/core/domain"

	"github.com/stretchr/testify/assert"
)

func TestGetLatest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := domain.ExchangeResponse{
			Base:  "USD",
			Date:  domain.CustomDate(time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC)),
			Rates: map[string]float64{"INR": 82.5, "EUR": 0.9},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	api := NewFrankFurterAPI(server.URL+"/", "2006-01-02")
	resp, err := api.GetLatest("USD", []string{"INR", "EUR"})
	assert.NoError(t, err)
	assert.Equal(t, "USD", resp.Base)
	assert.Equal(t, 82.5, resp.Rates["INR"])
	assert.Equal(t, 0.9, resp.Rates["EUR"])
	assert.Equal(t, time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC), resp.Date.ToTime())
}

func TestGetLatest_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer server.Close()

	api := NewFrankFurterAPI(server.URL+"/", "2006-01-02")
	resp, err := api.GetLatest("USD", []string{"INR"})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGetHistoricalTimeSeries_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := domain.HistoricalTimeSeriesRatesResponse{
			Base:      "USD",
			StartDate: "2024-05-01",
			EndDate:   "2024-05-07",
			Rates: map[string]map[string]float64{
				"2024-05-01": {"INR": 80.0},
				"2024-05-07": {"INR": 82.0},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	api := NewFrankFurterAPI(server.URL+"/", "2006-01-02")
	start := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC)
	resp, err := api.GetHistoricalTimeSeries("USD", []string{"INR"}, start, end)
	assert.NoError(t, err)
	assert.Equal(t, "USD", resp.Base)
	assert.Equal(t, 80.0, resp.Rates["2024-05-01"]["INR"])
	assert.Equal(t, 82.0, resp.Rates["2024-05-07"]["INR"])
}

func TestGetHistoricalTimeSeries_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer server.Close()

	api := NewFrankFurterAPI(server.URL+"/", "2006-01-02")
	start := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC)
	resp, err := api.GetHistoricalTimeSeries("USD", []string{"INR"}, start, end)
	assert.Error(t, err)
	assert.Nil(t, resp)
}
