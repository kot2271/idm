package info

import (
	"idm/inner/common"
	"idm/inner/web"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Controller struct {
	server *web.Server
	cfg    common.Config
	db     *sqlx.DB
	logger *common.Logger
}

func NewController(server *web.Server, cfg common.Config, db *sqlx.DB, logger *common.Logger) *Controller {
	return &Controller{
		server: server,
		cfg:    cfg,
		db:     db,
		logger: logger,
	}
}

type InfoResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func (c *Controller) RegisterRoutes() {
	c.logger.Info("registering info controller routes")
	// полный путь будет "/internal/info"
	c.server.GroupInternal.Get("/info", c.GetInfo)
	// полный путь будет "/internal/health"
	c.server.GroupInternal.Get("/health", c.GetHealth)
	c.logger.Info("info controller routes registered successfully")
}

// GetInfo получение информации о приложении
func (c *Controller) GetInfo(ctx *fiber.Ctx) error {
	c.logger.Debug("processing GetInfo request")

	response := &InfoResponse{
		Name:    c.cfg.AppName,
		Version: c.cfg.AppVersion,
	}

	c.logger.Info("GetInfo request processed successfully",
		zap.String("app_name", response.Name),
		zap.String("app_version", response.Version),
	)

	return ctx.Status(fiber.StatusOK).JSON(response)
}

// GetHealth проверка работоспособности приложения
func (c *Controller) GetHealth(ctx *fiber.Ctx) error {
	c.logger.Debug("processing GetHealth request")

	health := HealthResponse{
		Status:   "OK",
		Database: "OK",
	}

	// Проверка подключения к базе данных
	if c.db != nil {
		if err := c.db.Ping(); err != nil {
			c.logger.Error("database ping failed",
				zap.Error(err),
			)
			health.Status = "ERROR"
			health.Database = "ERROR"

			c.logger.Warn("GetHealth request completed with database error",
				zap.String("status", health.Status),
				zap.String("database", health.Database),
			)
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(&health)
		}
		c.logger.Debug("database ping successful")
	} else {
		c.logger.Error("database connection is nil")
		health.Status = "ERROR"
		health.Database = "NOT_CONNECTED"

		c.logger.Warn("GetHealth request completed with no database connection",
			zap.String("status", health.Status),
			zap.String("database", health.Database),
		)
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(&health)
	}

	c.logger.Info("GetHealth request processed successfully",
		zap.String("status", health.Status),
		zap.String("database", health.Database),
	)

	return ctx.Status(fiber.StatusOK).JSON(&health)
}
