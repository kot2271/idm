package web

import (
	"context"
	"fmt"
	"idm/inner/common"
	"idm/inner/testutils"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// Переменная для определения режима мокирования
var useKeycloakMocking = true

func SetupTestConfig() common.Config {
	wd, _ := os.Getwd()
	path := filepath.Join(wd, "..", "..", EnvFileName)

	var err = godotenv.Load(path)
	// если нет файла, то залогируем это и попробуем получить конфиг из переменных окружения
	if err != nil {
		fmt.Printf("Error loading .env file")
	}

	getEnv := func(key, fallback string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return fallback
	}

	keycloakJwkUrl := getEnv("KEYCLOAK_JWK_URL", DefaultJwkUrl)

	// Определяем, нужно ли использовать мокирование
	useKeycloakMocking = keycloakJwkUrl == DefaultJwkUrl

	return common.Config{
		DbDriverName:   "postgres",
		Dsn:            "localhost port=5432 user=wronguser password=wrongpass dbname=postgres sslmode=disable",
		AppName:        "test_app",
		AppVersion:     "1.0.0",
		LogLevel:       "DEBUG",
		LogDevelopMode: true,
		SslSert:        "ssl.cert",
		SslKey:         "ssl.key",
		KeycloakJwkUrl: keycloakJwkUrl,
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

	var accessToken string

	if useKeycloakMocking {
		// Используем мокированный токен
		accessToken = testutils.GenerateMockToken([]string{IdmUser})
	} else {
		// Используем реальный Keycloak
		cfg, err := testutils.LoadTestConfig("..", "")
		require.NoError(t, err)

		ctx := context.Background()
		token, err := testutils.GetKeycloakToken(
			ctx,
			cfg.Keycloak.Realm,
			cfg.Keycloak.ClientID,
			cfg.Keycloak.ClientSecret,
			cfg.Keycloak.Username2,
			cfg.Keycloak.Password,
		)
		require.NoError(t, err)
		require.NotEmpty(t, token)
		accessToken = token
	}

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
