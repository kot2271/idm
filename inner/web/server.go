package web

import (
	"idm/inner/common"
	"time"

	_ "idm/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/zap"
)

// структуа веб-сервера
type Server struct {
	App *fiber.App
	// группа публичного API
	GroupApi fiber.Router
	// группа публичного API первой версии
	GroupApiV1 fiber.Router
	// группа непубличного API
	GroupInternal fiber.Router
	// группа защищённого API (требует аутентификации)
	GroupApiV1Protected fiber.Router
	// группа для админов (требует роль IDM_ADMIN)
	GroupApiV1Admin fiber.Router
	// группа для пользователей (требует роль IDM_ADMIN или IDM_USER)
	GroupApiV1User fiber.Router
}

type AuthMiddlewareInterface interface {
	ProtectWithJwt() func(*fiber.Ctx) error
}

// функция-конструктор
func NewServer(logger *common.Logger) *Server {

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

	// Создаём защищённую группу с JWT middleware
	groupApiV1Protected := groupApiV1.Group("/")
	groupApiV1Protected.Use(AuthMiddleware(logger))

	// Создаём группу для админов (требует роль IDM_ADMIN)
	groupApiV1Admin := groupApiV1Protected.Group("/admin")
	groupApiV1Admin.Use(RequireRole(IdmAdmin, logger))

	// Создаём группу для пользователей (требует роль IDM_ADMIN или IDM_USER)
	groupApiV1User := groupApiV1Protected.Group("/")
	groupApiV1User.Use(RequireAnyRole([]string{IdmAdmin, IdmUser}, logger))

	return &Server{
		App:                 app,
		GroupApi:            groupApi,
		GroupApiV1:          groupApiV1,
		GroupInternal:       groupInternal,
		GroupApiV1Protected: groupApiV1Protected,
		GroupApiV1Admin:     groupApiV1Admin,
		GroupApiV1User:      groupApiV1User,
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
