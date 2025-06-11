package info

import (
	"idm/inner/common"
	"idm/inner/web"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

type Controller struct {
	server *web.Server
	cfg    common.Config
	db     *sqlx.DB
}

func NewController(server *web.Server, cfg common.Config, db *sqlx.DB) *Controller {
	return &Controller{
		server: server,
		cfg:    cfg,
		db:     db,
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
	// полный путь будет "/internal/info"
	c.server.GroupInternal.Get("/info", c.GetInfo)
	// полный путь будет "/internal/health"
	c.server.GroupInternal.Get("/health", c.GetHealth)
}

// GetInfo получение информации о приложении
func (c *Controller) GetInfo(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(&InfoResponse{
		Name:    c.cfg.AppName,
		Version: c.cfg.AppVersion,
	})
}

// GetHealth проверка работоспособности приложения
func (c *Controller) GetHealth(ctx *fiber.Ctx) error {
	health := HealthResponse{
		Status:   "OK",
		Database: "OK",
	}

	// Проверка подключения к базе данных
	if c.db != nil {
		if err := c.db.Ping(); err != nil {
			health.Status = "ERROR"
			health.Database = "ERROR"
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(&health)
		}
	} else {
		health.Status = "ERROR"
		health.Database = "NOT_CONNECTED"
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(&health)
	}

	return ctx.Status(fiber.StatusOK).JSON(&health)
}
