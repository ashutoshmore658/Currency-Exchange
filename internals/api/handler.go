package api

import (
	"currency-exchange/internals/core/domain"
	"currency-exchange/internals/service"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	rateService service.RateService
}

func NewHandler(rs service.RateService) *Handler {
	return &Handler{rateService: rs}
}

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	log.Printf("Error handling request: %v", err)

	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(ErrorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{
			Code:    http.StatusText(code),
			Message: message,
		},
	})
}

func (h *Handler) GetLatest(c *fiber.Ctx) error {
	baseCurrency := domain.Currency(strings.ToUpper(c.Query("base")))
	if baseCurrency == "" {
		return fiber.NewError(fiber.StatusBadRequest, "base query parameter is required")
	}

	symbolsStr := strings.ToUpper(c.Query("symbol"))

	if symbolsStr != "" {
		if len(strings.Split(symbolsStr, ",")) > 1 {
			return fiber.NewError(fiber.StatusBadRequest, "More than one target currencies provided, specify one !")
		}

	}

	rates, err := h.rateService.GetLatestRates(c.Context(), baseCurrency, domain.Currency(symbolsStr))
	if err != nil {
		return err
	}

	return c.JSON(rates)
}

func (h *Handler) Convert(c *fiber.Ctx) error {
	fromCurrency := domain.Currency(strings.ToUpper(c.Query("from")))
	toCurrency := domain.Currency(strings.ToUpper(c.Query("to")))
	amountStr := c.Query("amount")

	if fromCurrency == "" || toCurrency == "" || amountStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "from, to, and amount query parameters are required")
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "amount must be a positive number")
	}

	dateStr := c.Query("date")
	var conversionDate *time.Time
	if dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid `date` format, expected YYYY-MM-DD")
		}
		conversionDate = &parsedDate
	} else {
		conversionDate = nil
	}

	req := domain.ConversionRequest{
		From:   fromCurrency,
		To:     toCurrency,
		Amount: amount,
		Date:   conversionDate,
	}

	result, err := h.rateService.Convert(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func (h *Handler) GetHistorical(c *fiber.Ctx) error {
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	baseCurrency := domain.Currency(strings.ToUpper(c.Query("base")))

	if baseCurrency == "" {
		return fiber.NewError(fiber.StatusBadRequest, "`base` query parameter is required")
	}

	if startDate == "" && endDate == "" {
		return fiber.NewError(fiber.StatusBadRequest, "at least one of `startDate` or `endDate` query parameters is required to get historical time series data")
	}

	if startDate == "" {
		startDate = endDate
	} else if endDate == "" {
		endDate = startDate
	}

	symbolsStr := strings.ToUpper(c.Query("symbol"))

	if symbolsStr != "" {
		if len(strings.Split(symbolsStr, ",")) > 1 {
			return fiber.NewError(fiber.StatusBadRequest, "More than one target currencies provided, specify one !")
		}

	}

	rates, err := h.rateService.GetHistoricalRates(c.Context(), startDate, endDate, baseCurrency, domain.Currency(symbolsStr))
	if err != nil {
		return err
	}

	return c.JSON(rates)
}
