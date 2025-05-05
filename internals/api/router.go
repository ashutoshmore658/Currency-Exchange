package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func SetupRouter(app *fiber.App, handler *Handler) {

	// Middleware
	app.Use(logger.New())

	// Routes
	v1 := app.Group("/v1")
	{
		v1.Get("/latest", handler.GetLatest)
		v1.Get("/convert", handler.Convert)
		v1.Get("/historical", handler.GetHistorical)
	}

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "UP"})
	})
}
