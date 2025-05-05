package main

import (
	"context"
	"currency-exchange/internals/adapter/exchangerateapi"
	"currency-exchange/internals/api"
	"currency-exchange/internals/config"
	"currency-exchange/internals/repository"
	"currency-exchange/internals/service"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	log.Println("Starting Exchange Rate Service...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	apiClient := exchangerateapi.NewClient()
	rateRepo := repository.NewCachedRateRepository(apiClient)
	rateService := service.NewRateService(rateRepo, 90)
	apiHandler := api.NewHandler(rateService)

	app := fiber.New(fiber.Config{
		AppName:      "Exchange Rate Service",
		ErrorHandler: api.ErrorHandler,
	})

	app.Use(logger.New())

	api.SetupRouter(app, apiHandler)

	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := app.Listen(":" + cfg.ServerPort); err != nil {
			log.Fatalf("Could not start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server exited gracefully")
}
