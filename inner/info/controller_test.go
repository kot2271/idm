package info

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"idm/inner/common"
	"idm/inner/web"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Создаем тестовый сервер
func setupTestServer() (*fiber.App, *web.Server) {
	app := fiber.New()
	server := &web.Server{
		App:           app,
		GroupInternal: app.Group("/internal"),
	}
	return app, server
}

func TestController_GetInfo_Success(t *testing.T) {
	app, server := setupTestServer()

	cfg := common.Config{
		DbDriverName: "postgres",
		Dsn:          "test-dsn",
		AppName:      "test-app",
		AppVersion:   "1.0.0",
	}

	logger := common.NewLogger(cfg)

	controller := NewController(server, cfg, nil, logger)
	controller.RegisterRoutes()

	req := httptest.NewRequest("GET", "/internal/info", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var infoResponse InfoResponse
	err = json.Unmarshal(body, &infoResponse)
	require.NoError(t, err)

	assert.Equal(t, "test-app", infoResponse.Name)
	assert.Equal(t, "1.0.0", infoResponse.Version)
}

func TestController_GetHealth_WithHealthyDB(t *testing.T) {
	app, server := setupTestServer()

	cfg := common.Config{
		DbDriverName: "postgres",
		Dsn:          "test-dsn",
		AppName:      "test-app",
		AppVersion:   "1.0.0",
	}

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)

	defer func() {
		_ = db.Close()
	}()

	// sqlx.DB из sql.DB
	sqlxDB := sqlx.NewDb(db, "postgres")

	mock.ExpectPing()

	logger := common.NewLogger(cfg)

	controller := NewController(server, cfg, sqlxDB, logger)
	controller.RegisterRoutes()

	req := httptest.NewRequest("GET", "/internal/health", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResponse HealthResponse
	err = json.Unmarshal(body, &healthResponse)
	require.NoError(t, err)

	assert.Equal(t, "OK", healthResponse.Status)
	assert.Equal(t, "OK", healthResponse.Database)

	// Проверка, что метод Ping был вызван
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestController_GetHealth_WithUnhealthyDB(t *testing.T) {
	app, server := setupTestServer()

	cfg := common.Config{
		DbDriverName: "postgres",
		Dsn:          "test-dsn",
		AppName:      "test-app",
		AppVersion:   "1.0.0",
	}

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)

	defer func() {
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "postgres")

	mock.ExpectPing().WillReturnError(errors.New("database not available"))

	logger := common.NewLogger(cfg)

	controller := NewController(server, cfg, sqlxDB, logger)
	controller.RegisterRoutes()

	req := httptest.NewRequest("GET", "/internal/health", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResponse HealthResponse
	err = json.Unmarshal(body, &healthResponse)
	require.NoError(t, err)

	assert.Equal(t, "ERROR", healthResponse.Status)
	assert.Equal(t, "ERROR", healthResponse.Database)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestController_GetHealth_WithNilDB(t *testing.T) {
	app, server := setupTestServer()

	cfg := common.Config{
		DbDriverName: "postgres",
		Dsn:          "test-dsn",
		AppName:      "test-app",
		AppVersion:   "1.0.0",
	}

	logger := common.NewLogger(cfg)

	controller := NewController(server, cfg, nil, logger)
	controller.RegisterRoutes()

	req := httptest.NewRequest("GET", "/internal/health", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResponse HealthResponse
	err = json.Unmarshal(body, &healthResponse)
	require.NoError(t, err)

	assert.Equal(t, "ERROR", healthResponse.Status)
	assert.Equal(t, "NOT_CONNECTED", healthResponse.Database)
}
