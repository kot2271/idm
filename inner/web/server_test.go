package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestRecoverMiddleware(t *testing.T) {
	server := NewServer()
	server.App.Get("/panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	resp, err := server.App.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestRequestIDMiddleware(t *testing.T) {
	server := NewServer()
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
	server := NewServer()
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
	server := NewServer()
	server.GroupApiV1.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("api v1 ok")
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
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
