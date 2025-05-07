package service

import (
	"context"
	"currency-exchange/internals/core/domain"
	"currency-exchange/internals/repository"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

var (
	ErrCurrencyNotSupported = errors.New("currency not supported")
	ErrRateNotFound         = errors.New("exchange rate not found")
)

// RateService defines the business logic for exchange rates.
type RateService interface {
	GetLatestRate(ctx context.Context, base, target domain.Currency) (float64, time.Time, error)
	Convert(ctx context.Context, req domain.ConversionRequest) (*domain.ConversionResult, error)
	GetHistoricalRate(ctx context.Context, onDate time.Time, base, target domain.Currency) (float64, error)
	GetLatestRates(ctx context.Context, base domain.Currency, targets domain.Currency) (*domain.LatestRates, error)
	GetHistoricalRates(ctx context.Context, startDate string, endDate string, base domain.Currency, targets domain.Currency) (*domain.HistoricalRates, error)
	GetSupportedCurrencies() []string
	ValidateCurrencies(currency domain.Currency) error
}

type rateServiceImpl struct {
	repo             repository.RateRepository
	historyDaysLimit int
}

func NewRateService(repo repository.RateRepository, historyDaysLimit int) RateService {
	return &rateServiceImpl{
		repo:             repo,
		historyDaysLimit: historyDaysLimit,
	}
}

func (s *rateServiceImpl) GetSupportedCurrencies() []string {
	keys := make([]string, 0, len(domain.SupportedCurrencies))
	for k := range domain.SupportedCurrencies {
		keys = append(keys, string(k))
	}
	return keys
}

func (s *rateServiceImpl) ValidateCurrencies(currency domain.Currency) error {
	if !currency.IsSupported() {
		return fmt.Errorf("%w: %s", ErrCurrencyNotSupported, currency)
	}

	return nil
}

func (s *rateServiceImpl) validateDate(dateStr string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fiber.NewError(fiber.StatusBadRequest, "invalid date format please format the date in yyyy-mm-dd")
	}

	oldestAllowedDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -s.historyDaysLimit)
	if date.Before(oldestAllowedDate) {
		return time.Time{}, fiber.NewError(fiber.StatusBadRequest, "requested date is older than 90 days")
	}

	if date.After(time.Now().UTC().Truncate(24 * time.Hour)) {
		return time.Time{}, fiber.NewError(fiber.StatusBadRequest, "historical date can not be in future")
	}

	return date, nil
}

func (s *rateServiceImpl) GetLatestRate(ctx context.Context, base, target domain.Currency) (float64, time.Time, error) {

	if base == target {
		return 1.0, time.Now().UTC(), nil // Rate to self is always 1
	}

	rates, timestamp, err := s.repo.GetLatestRates(ctx, base, target)
	if err != nil {
		return 0, time.Time{}, err
	}

	rate, ok := rates[target]
	if !ok {
		log.Printf("Rate not found in repository result for %s -> %s", base, target)
		return 0, time.Time{}, ErrRateNotFound
	}

	return rate, timestamp, nil
}

func (s *rateServiceImpl) Convert(ctx context.Context, req domain.ConversionRequest) (*domain.ConversionResult, error) {
	var err error
	if req.From == req.To {
		return nil, fiber.NewError(fiber.StatusBadRequest, "from and to currencies cannot be the same for conversion")
	}
	var rate float64
	if req.Date == nil {
		rate, _, err = s.GetLatestRate(ctx, req.From, req.To)
	} else {
		rate, err = s.GetHistoricalRate(ctx, *req.Date, req.From, req.To)
	}
	if err != nil {
		return nil, fmt.Errorf("could not get rate for conversion: %w", err)
	}

	convertedAmount := req.Amount * rate

	return &domain.ConversionResult{
		From:            req.From,
		To:              req.To,
		OriginalAmount:  req.Amount,
		ConvertedAmount: convertedAmount,
		Rate:            rate,
		Date:            req.Date,
	}, nil
}

func (s *rateServiceImpl) GetHistoricalRate(ctx context.Context, onDate time.Time, base, target domain.Currency) (float64, error) {

	if base == target {
		return 1.0, nil // Rate to self is always 1
	}

	currencyRates, err := s.repo.GetHistoricalRates(ctx, onDate, onDate, base, target)
	if err != nil {
		return 0, err
	}

	rate, ok := currencyRates[onDate]
	if !ok {
		log.Printf("Historical rate not found in repository result for %s -> %s on %s", base, target, onDate)
		return 0, ErrRateNotFound
	}

	return rate, nil
}

func (s *rateServiceImpl) GetLatestRates(ctx context.Context, base domain.Currency, target domain.Currency) (*domain.LatestRates, error) {

	rates, timestamp, err := s.repo.GetLatestRates(ctx, base, target)
	if err != nil {
		return nil, err
	}

	rates[base] = 1.0

	return &domain.LatestRates{
		Base:      base,
		Rates:     rates,
		Timestamp: timestamp.Unix(),
	}, nil
}

func (s *rateServiceImpl) GetHistoricalRates(ctx context.Context, startDate string, endDate string, base domain.Currency, target domain.Currency) (*domain.HistoricalRates, error) {
	convStartDate, err := s.validateDate(startDate)
	if err != nil {
		return nil, err
	}

	convEndDate, err := s.validateDate(endDate)
	if err != nil {
		return nil, err
	}

	rates, err := s.repo.GetHistoricalRates(ctx, convStartDate, convEndDate, base, target)
	if err != nil {
		return nil, err
	}

	return &domain.HistoricalRates{
		Base:   base,
		Rates:  rates,
		Amount: 1.0,
		Target: target,
	}, nil
}
