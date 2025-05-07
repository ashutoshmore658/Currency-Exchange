package service

import (
	"context"
	"currency-exchange/internals/core/domain"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- Mock Repository ---

type MockRateRepository struct {
	LatestRatesResp     map[domain.Currency]float64
	LatestRatesTime     time.Time
	LatestRatesErr      error
	HistoricalRatesResp map[time.Time]float64
	HistoricalRatesErr  error
}

func (m *MockRateRepository) GetLatestRates(ctx context.Context, base, target domain.Currency) (map[domain.Currency]float64, time.Time, error) {
	return m.LatestRatesResp, m.LatestRatesTime, m.LatestRatesErr
}
func (m *MockRateRepository) GetHistoricalRates(ctx context.Context, startDate, endDate time.Time, base, target domain.Currency) (map[time.Time]float64, error) {
	return m.HistoricalRatesResp, m.HistoricalRatesErr
}

func ptrTime(t time.Time) *time.Time { return &t }

// --- Tests ---

func TestGetSupportedCurrencies(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	currencies := svc.GetSupportedCurrencies()
	assert.Contains(t, currencies, "USD")
	assert.Contains(t, currencies, "INR")
	assert.Len(t, currencies, 5)
}

func TestValidateCurrencies_Supported(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	err := svc.ValidateCurrencies("USD")
	assert.NoError(t, err)
}

func TestValidateCurrencies_Unsupported(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	err := svc.ValidateCurrencies("FOO")
	assert.ErrorIs(t, err, ErrCurrencyNotSupported)
}

func TestValidateDate_Valid(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	dateStr := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	date, err := svc.(*rateServiceImpl).validateDate(dateStr)
	assert.NoError(t, err)
	assert.Equal(t, dateStr, date.Format("2006-01-02"))
}

func TestValidateDate_TooOld(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	dateStr := time.Now().AddDate(0, 0, -100).Format("2006-01-02")
	_, err := svc.(*rateServiceImpl).validateDate(dateStr)
	assert.ErrorIs(t, err, ErrDateTooOld)
}

func TestValidateDate_Future(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	dateStr := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	_, err := svc.(*rateServiceImpl).validateDate(dateStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "future")
}

func TestValidateDate_InvalidFormat(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	_, err := svc.(*rateServiceImpl).validateDate("2024-13-40")
	assert.ErrorIs(t, err, ErrInvalidDateFormat)
}

func TestGetLatestRate_SameCurrency(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	rate, ts, err := svc.GetLatestRate(context.Background(), "USD", "USD")
	assert.NoError(t, err)
	assert.Equal(t, 1.0, rate)
	assert.WithinDuration(t, time.Now().UTC(), ts, time.Second)
}

func TestGetLatestRate_RepoError(t *testing.T) {
	mockRepo := &MockRateRepository{LatestRatesErr: errors.New("repo error")}
	svc := NewRateService(mockRepo, 90)
	_, _, err := svc.GetLatestRate(context.Background(), "USD", "INR")
	assert.Error(t, err)
}

func TestGetLatestRate_RateNotFound(t *testing.T) {
	mockRepo := &MockRateRepository{
		LatestRatesResp: map[domain.Currency]float64{"EUR": 0.9},
		LatestRatesTime: time.Now(),
	}
	svc := NewRateService(mockRepo, 90)
	_, _, err := svc.GetLatestRate(context.Background(), "USD", "INR")
	assert.ErrorIs(t, err, ErrRateNotFound)
}

func TestGetLatestRate_Success(t *testing.T) {
	mockRepo := &MockRateRepository{
		LatestRatesResp: map[domain.Currency]float64{"INR": 82.5},
		LatestRatesTime: time.Now(),
	}
	svc := NewRateService(mockRepo, 90)
	rate, ts, err := svc.GetLatestRate(context.Background(), "USD", "INR")
	assert.NoError(t, err)
	assert.Equal(t, 82.5, rate)
	assert.WithinDuration(t, time.Now(), ts, time.Second)
}

func TestConvert_InvalidAmount(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	req := domain.ConversionRequest{From: "USD", To: "INR", Amount: -10}
	_, err := svc.Convert(context.Background(), req)
	assert.ErrorIs(t, err, ErrInvalidAmount)
}

func TestConvert_SameCurrency(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	req := domain.ConversionRequest{From: "USD", To: "USD", Amount: 10}
	_, err := svc.Convert(context.Background(), req)
	assert.ErrorIs(t, err, ErrSameCurrency)
}

func TestConvert_LatestRate_Success(t *testing.T) {
	mockRepo := &MockRateRepository{
		LatestRatesResp: map[domain.Currency]float64{"INR": 80.0},
		LatestRatesTime: time.Now(),
	}
	svc := NewRateService(mockRepo, 90)
	req := domain.ConversionRequest{From: "USD", To: "INR", Amount: 10}
	res, err := svc.Convert(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, 800.0, res.ConvertedAmount)
	assert.Equal(t, 80.0, res.Rate)
}

func TestConvert_HistoricalRate_Success(t *testing.T) {
	date := time.Now().AddDate(0, 0, -5).Truncate(24 * time.Hour)
	mockRepo := &MockRateRepository{
		HistoricalRatesResp: map[time.Time]float64{date: 75.0},
	}
	svc := NewRateService(mockRepo, 90)
	req := domain.ConversionRequest{From: "USD", To: "INR", Amount: 10, Date: &date}
	res, err := svc.Convert(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, 750.0, res.ConvertedAmount)
	assert.Equal(t, 75.0, res.Rate)
}

func TestConvert_RepoError(t *testing.T) {
	mockRepo := &MockRateRepository{LatestRatesErr: errors.New("repo error")}
	svc := NewRateService(mockRepo, 90)
	req := domain.ConversionRequest{From: "USD", To: "INR", Amount: 10}
	_, err := svc.Convert(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not get rate for conversion")
}

func TestGetHistoricalRate_SameCurrency(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	rate, err := svc.GetHistoricalRate(context.Background(), time.Now(), "USD", "USD")
	assert.NoError(t, err)
	assert.Equal(t, 1.0, rate)
}

func TestGetHistoricalRate_RepoError(t *testing.T) {
	mockRepo := &MockRateRepository{HistoricalRatesErr: errors.New("repo error")}
	svc := NewRateService(mockRepo, 90)
	_, err := svc.GetHistoricalRate(context.Background(), time.Now(), "USD", "INR")
	assert.Error(t, err)
}

func TestGetHistoricalRate_RateNotFound(t *testing.T) {
	date := time.Now().Truncate(24 * time.Hour)
	mockRepo := &MockRateRepository{
		HistoricalRatesResp: map[time.Time]float64{},
	}
	svc := NewRateService(mockRepo, 90)
	_, err := svc.GetHistoricalRate(context.Background(), date, "USD", "INR")
	assert.ErrorIs(t, err, ErrRateNotFound)
}

func TestGetHistoricalRate_Success(t *testing.T) {
	date := time.Now().Truncate(24 * time.Hour)
	mockRepo := &MockRateRepository{
		HistoricalRatesResp: map[time.Time]float64{date: 81.0},
	}
	svc := NewRateService(mockRepo, 90)
	rate, err := svc.GetHistoricalRate(context.Background(), date, "USD", "INR")
	assert.NoError(t, err)
	assert.Equal(t, 81.0, rate)
}

func TestGetLatestRates_RepoError(t *testing.T) {
	mockRepo := &MockRateRepository{LatestRatesErr: errors.New("repo error")}
	svc := NewRateService(mockRepo, 90)
	_, err := svc.GetLatestRates(context.Background(), "USD", "INR")
	assert.Error(t, err)
}

func TestGetLatestRates_Success(t *testing.T) {
	mockRepo := &MockRateRepository{
		LatestRatesResp: map[domain.Currency]float64{"INR": 79.0},
		LatestRatesTime: time.Now(),
	}
	svc := NewRateService(mockRepo, 90)
	res, err := svc.GetLatestRates(context.Background(), "USD", "INR")
	assert.NoError(t, err)
	assert.Equal(t, "USD", string(res.Base))
	assert.Equal(t, 79.0, res.Rates["INR"])
	assert.Equal(t, 1.0, res.Rates["USD"])
}

func TestGetHistoricalRates_Valid(t *testing.T) {
	date := time.Now().Truncate(24 * time.Hour)
	mockRepo := &MockRateRepository{
		HistoricalRatesResp: map[time.Time]float64{date: 77.0},
	}
	svc := NewRateService(mockRepo, 90)
	res, err := svc.GetHistoricalRates(context.Background(), date.Format("2006-01-02"), date.Format("2006-01-02"), "USD", "INR")
	assert.NoError(t, err)
	assert.Equal(t, "USD", string(res.Base))
	assert.Equal(t, 77.0, res.Rates[date])
	assert.Equal(t, "INR", string(res.Target))
}

func TestGetHistoricalRates_InvalidStartDate(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	_, err := svc.GetHistoricalRates(context.Background(), "invalid", "2024-05-01", "USD", "INR")
	assert.ErrorIs(t, err, ErrInvalidDateFormat)
}

func TestGetHistoricalRates_InvalidEndDate(t *testing.T) {
	svc := NewRateService(&MockRateRepository{}, 90)
	start := time.Now().Format("2006-01-02")
	_, err := svc.GetHistoricalRates(context.Background(), start, "invalid", "USD", "INR")
	assert.ErrorIs(t, err, ErrInvalidDateFormat)
}

func TestGetHistoricalRates_RepoError(t *testing.T) {
	date := time.Now().Truncate(24 * time.Hour)
	mockRepo := &MockRateRepository{HistoricalRatesErr: errors.New("repo error")}
	svc := NewRateService(mockRepo, 90)
	_, err := svc.GetHistoricalRates(context.Background(), date.Format("2006-01-02"), date.Format("2006-01-02"), "USD", "INR")
	assert.Error(t, err)
}
