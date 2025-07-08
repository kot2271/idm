package web

import (
	"context"
	"idm/inner/common"
	"idm/inner/testutils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func SetupTestConfig() common.Config {
	return common.Config{
		DbDriverName:   "postgres",
		Dsn:            "localhost port=5432 user=wronguser password=wrongpass dbname=postgres sslmode=disable",
		AppName:        "test_app",
		AppVersion:     "1.0.0",
		LogLevel:       "DEBUG",
		LogDevelopMode: true,
		SslSert:        "ssl.cert",
		SslKey:         "ssl.key",
		KeycloakJwkUrl: "http://localhost:9990/realms/idm/protocol/openid-connect/certs",
	}
}

func SetupTestLogger() *common.Logger {
	cfg := SetupTestConfig()
	logger := common.NewLogger(cfg)
	return logger
}

func TestRecoverMiddleware(t *testing.T) {
	logger := SetupTestLogger()
	server := NewServer(logger)
	server.App.Get("/panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	resp, err := server.App.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestRequestIDMiddleware(t *testing.T) {
	logger := SetupTestLogger()
	server := NewServer(logger)
	server.App.Get("/id", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/id", nil)
	resp, err := server.App.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-Id"))
}

func TestInternalGroupMiddleware(t *testing.T) {
	logger := SetupTestLogger()
	server := NewServer(logger)
	server.GroupInternal.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("internal ok")
	})

	req := httptest.NewRequest("GET", "/internal/test", nil)
	resp, err := server.App.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "true", resp.Header.Get("X-Internal-API"))
}

func TestApiV1GroupMiddleware(t *testing.T) {
	logger := SetupTestLogger()
	server := NewServer(logger)
	server.GroupApiV1.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("api v1 ok")
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.Background()
	cfg, _ := testutils.LoadTestConfig("..", "")
	accessToken, err := testutils.GetKeycloakToken(
		ctx,
		cfg.Keycloak.Realm,
		cfg.Keycloak.ClientID,
		cfg.Keycloak.ClientSecret,
		cfg.Keycloak.Username2,
		cfg.Keycloak.Password,
	)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := server.App.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "v1", resp.Header.Get("X-API-Version"))
}

func TestCustomMiddleware_LogsRequest(t *testing.T) {
	logger := zaptest.NewLogger(t)
	app := fiber.New()
	app.Use(CustomMiddleware(logger))
	app.Get("/log", func(c *fiber.Ctx) error {
		return c.SendString("logged")
	})

	req := httptest.NewRequest("GET", "/log", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
