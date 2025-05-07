package api

import (
	"context"
	"currency-exchange/internals/core/domain"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// --- Mock Service Implementation ---

type MockRateService struct {
	LatestRatesResp    *domain.LatestRates
	LatestRatesErr     error
	ConversionResult   *domain.ConversionResult
	ConversionErr      error
	HistoricalRates    *domain.HistoricalRates
	HistoricalRatesErr error
	ValidateErr        error
}

func (m *MockRateService) GetLatestRate(ctx context.Context, base, target domain.Currency) (float64, time.Time, error) {
	if m.LatestRatesErr != nil {
		return 0, time.Time{}, m.LatestRatesErr
	}
	return 82.5, time.Now(), nil
}
func (m *MockRateService) Convert(ctx context.Context, req domain.ConversionRequest) (*domain.ConversionResult, error) {
	if m.ConversionErr != nil {
		return nil, m.ConversionErr
	}
	return m.ConversionResult, nil
}
func (m *MockRateService) GetHistoricalRate(ctx context.Context, onDate time.Time, base, target domain.Currency) (float64, error) {
	return 80.0, nil
}
func (m *MockRateService) GetLatestRates(ctx context.Context, base domain.Currency, target domain.Currency) (*domain.LatestRates, error) {
	if m.LatestRatesErr != nil {
		return nil, m.LatestRatesErr
	}
	return m.LatestRatesResp, nil
}
func (m *MockRateService) GetHistoricalRates(ctx context.Context, startDate, endDate string, base domain.Currency, target domain.Currency) (*domain.HistoricalRates, error) {
	if m.HistoricalRatesErr != nil {
		return nil, m.HistoricalRatesErr
	}
	return m.HistoricalRates, nil
}
func (m *MockRateService) GetSupportedCurrencies() []string {
	return []string{"USD", "INR", "EUR", "JPY", "GBP"}
}
func (m *MockRateService) ValidateCurrencies(currency domain.Currency) error {
	return m.ValidateErr
}

// --- Helper to setup Fiber app with routes ---

func setupTestApp(mock *MockRateService) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})
	h := NewHandler(mock)
	app.Get("/v1/latest", h.GetLatest)
	app.Get("/v1/convert", h.Convert)
	app.Get("/v1/historical", h.GetHistorical)
	return app
}

// --- Tests for /v1/latest ---

func TestGetLatest_Success(t *testing.T) {
	mock := &MockRateService{
		LatestRatesResp: &domain.LatestRates{
			Base:  "USD",
			Rates: map[domain.Currency]float64{"INR": 82.5},
		},
	}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/latest?base=USD&symbol=INR", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	var result domain.LatestRates
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "USD", string(result.Base))
	assert.Equal(t, 82.5, result.Rates["INR"])
}

func TestGetLatest_MissingBase(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/latest?symbol=INR", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetLatest_MissingSymbol(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/latest?base=USD", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetLatest_MultipleSymbols(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/latest?base=USD&symbol=INR,EUR", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetLatest_ValidationError(t *testing.T) {
	mock := &MockRateService{ValidateErr: errors.New("currency not supported")}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/latest?base=FOO&symbol=INR", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetLatest_ServiceError(t *testing.T) {
	mock := &MockRateService{LatestRatesErr: errors.New("service error")}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/latest?base=USD&symbol=INR", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 500, resp.StatusCode)
}

// --- Tests for /v1/convert ---

func TestConvert_Success(t *testing.T) {
	mock := &MockRateService{
		ConversionResult: &domain.ConversionResult{
			From:            "USD",
			To:              "INR",
			OriginalAmount:  100,
			ConvertedAmount: 8250,
			Rate:            82.5,
		},
	}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/convert?from=USD&to=INR&amount=100", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	var result domain.ConversionResult
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "USD", string(result.From))
	assert.Equal(t, "INR", string(result.To))
	assert.Equal(t, 8250.0, result.ConvertedAmount)
}

func TestConvert_MissingParams(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/convert?from=USD&to=INR", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestConvert_InvalidAmount(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/convert?from=USD&to=INR&amount=-5", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestConvert_ValidationError(t *testing.T) {
	mock := &MockRateService{ValidateErr: errors.New("currency not supported")}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/convert?from=FOO&to=INR&amount=10", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestConvert_ServiceError(t *testing.T) {
	mock := &MockRateService{ConversionErr: errors.New("conversion error")}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/convert?from=USD&to=INR&amount=10", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestConvert_DateParam_Success(t *testing.T) {
	mock := &MockRateService{
		ConversionResult: &domain.ConversionResult{
			From:            "USD",
			To:              "INR",
			OriginalAmount:  100,
			ConvertedAmount: 8000,
			Rate:            80.0,
			Date:            ptrTime(time.Now().AddDate(0, 0, -10)),
		},
	}
	app := setupTestApp(mock)
	date := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/convert?from=USD&to=INR&amount=100&date=%s", date), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	var result domain.ConversionResult
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, 8000.0, result.ConvertedAmount)
}

func TestConvert_InvalidDate(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/convert?from=USD&to=INR&amount=100&date=2025-13-01", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

// --- Tests for /v1/historical ---

func TestGetHistorical_Success(t *testing.T) {
	mock := &MockRateService{
		HistoricalRates: &domain.HistoricalRates{
			Base:   "USD",
			Target: "INR",
			Rates:  map[time.Time]float64{time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour): 80.0},
		},
	}
	app := setupTestApp(mock)
	date := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/historical?base=USD&symbol=INR&startDate=%s", date), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	var result domain.HistoricalRates
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "USD", string(result.Base))
	assert.Equal(t, "INR", string(result.Target))
}

func TestGetHistorical_MissingBase(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/historical?symbol=INR&startDate=2024-05-01", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetHistorical_MissingSymbol(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/historical?base=USD&startDate=2024-05-01", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetHistorical_MissingDates(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/historical?base=USD&symbol=INR", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetHistorical_MultipleSymbols(t *testing.T) {
	mock := &MockRateService{}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/historical?base=USD&symbol=INR,EUR&startDate=2024-05-01", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetHistorical_ValidationError(t *testing.T) {
	mock := &MockRateService{ValidateErr: errors.New("currency not supported")}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/historical?base=FOO&symbol=INR&startDate=2024-05-01", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetHistorical_ServiceError(t *testing.T) {
	mock := &MockRateService{HistoricalRatesErr: errors.New("repo error")}
	app := setupTestApp(mock)
	req := httptest.NewRequest("GET", "/v1/historical?base=USD&symbol=INR&startDate=2024-05-01", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 500, resp.StatusCode)
}

func ptrTime(t time.Time) *time.Time { return &t }
