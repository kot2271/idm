package web

import (
	"time"

	_ "idm/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/swagger"
	"go.uber.org/zap"
)

// структуа веб-сервера
type Server struct {
	App *fiber.App
	// группа публичного API
	GroupApiV1 fiber.Router
	// группа непубличного API
	GroupInternal fiber.Router
}

// функция-конструктор
func NewServer() *Server {

	// создаём новый веб-вервер
	app := fiber.New()

	// Middleware для восстановления от паники
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Middleware для добавления уникального ID к каждому запросу
	app.Use(requestid.New())

	groupInternal := app.Group("/internal")

	// Middleware для внутренних маршрутов
	groupInternal.Use(func(c *fiber.Ctx) error {
		// дополнительная проверка для внутренних маршрутов
		c.Set("X-Internal-API", "true")
		return c.Next()
	})

	// создаём группу "/api"
	groupApi := app.Group("/api")

	// создаём подгруппу "api/v1"
	groupApiV1 := groupApi.Group("/v1")

	// Middleware для API v1
	groupApiV1.Use(func(c *fiber.Ctx) error {
		// Добавляем заголовок версии API
		c.Set("X-API-Version", "v1")
		return c.Next()
	})

	// добавляем маршрут для Swagger UI
	app.Get("/swagger/*", swagger.HandlerDefault) // default

	return &Server{
		App:           app,
		GroupApiV1:    groupApiV1,
		GroupInternal: groupInternal,
	}
}

func CustomMiddleware(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Логирование начала запроса
		logger.Info("Request started",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
		)

		// Выполняется следующий handler
		err := c.Next()

		// Логирование завершения запроса
		duration := time.Since(start)
		logger.Info("Request completed",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("duration", duration),
		)

		return err
	}
}
