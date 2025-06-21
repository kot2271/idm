package role

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"idm/inner/common"
	"idm/inner/web"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock для сервиса
type MockService struct {
	mock.Mock
}

func (m *MockService) FindById(ctx context.Context, id int64) (Response, error) {
	args := m.Called(id)
	return args.Get(0).(Response), args.Error(1)
}

func (m *MockService) CreateRole(ctx context.Context, request CreateRequest) (int64, error) {
	args := m.Called(request)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockService) DeleteById(ctx context.Context, id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockService) DeleteByIds(ctx context.Context, ids []int64) error {
	args := m.Called(ids)
	return args.Error(0)
}

func (m *MockService) FindAll(ctx context.Context) ([]Response, error) {
	args := m.Called()
	return args.Get(0).([]Response), args.Error(1)
}

func (m *MockService) FindByIds(ctx context.Context, ids []int64) ([]Response, error) {
	args := m.Called(ids)
	return args.Get(0).([]Response), args.Error(1)
}

// Вспомогательные функции для создания Fiber app
func setupTestApp() (*fiber.App, *MockService) {
	app := fiber.New()
	mockService := &MockService{}

	// Создаем mock web.Server
	server := &web.Server{
		GroupApiV1: app.Group("/api/v1"),
	}

	cfg := common.Config{
		DbDriverName:   "postgres",
		Dsn:            "localhost port=5432 user=wronguser password=wrongpass dbname=postgres sslmode=disable",
		AppName:        "test_app",
		AppVersion:     "1.0.0",
		LogLevel:       "DEBUG",
		LogDevelopMode: true,
	}

	logger := common.NewLogger(cfg)

	controller := NewController(server, mockService, logger)
	controller.RegisterRoutes()

	return app, mockService
}

func TestController_CreateRole_Success(t *testing.T) {
	app, mockService := setupTestApp()

	requestBody := CreateRequest{
		Name:        "Test Role",
		Description: "Test Description",
		Status:      true,
		ParentId:    nil,
	}

	expectedRoleId := int64(123)

	mockService.On("CreateRole", requestBody).Return(expectedRoleId, nil)

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response struct {
		Success bool           `json:"success"`
		Message string         `json:"error"`
		Data    map[string]any `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	id, ok := response.Data["id"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(expectedRoleId), id)

	mockService.AssertExpectations(t)
}

func TestController_CreateRole_InvalidJSON(t *testing.T) {
	app, mockService := setupTestApp()

	// Отправляем невалидный JSON
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotEmpty(t, response.Message)

	// Mock не должен был вызываться
	mockService.AssertNotCalled(t, "CreateRole")
}

func TestController_CreateRole_ValidationError(t *testing.T) {
	app, mockService := setupTestApp()

	requestBody := CreateRequest{
		Name:        "Test Role",
		Description: "Test Description",
		Status:      true,
		ParentId:    nil,
	}

	validationErr := common.RequestValidationError{
		Message: "Name is required",
	}

	mockService.On("CreateRole", requestBody).Return(int64(0), validationErr)

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, validationErr.Message, response.Message)

	mockService.AssertExpectations(t)
}

func TestController_CreateRole_AlreadyExistsError(t *testing.T) {
	app, mockService := setupTestApp()

	requestBody := CreateRequest{
		Name:        "Existing Role",
		Description: "Test Description",
		Status:      true,
		ParentId:    nil,
	}

	alreadyExistsErr := common.AlreadyExistsError{
		Message: "Role with this name already exists",
	}

	mockService.On("CreateRole", requestBody).Return(int64(0), alreadyExistsErr)

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, alreadyExistsErr.Message, response.Message)

	mockService.AssertExpectations(t)
}

func TestController_CreateRole_InternalServerError(t *testing.T) {
	app, mockService := setupTestApp()

	requestBody := CreateRequest{
		Name:        "Test Role",
		Description: "Test Description",
		Status:      true,
		ParentId:    nil,
	}

	internalErr := errors.New("Internal server error")

	mockService.On("CreateRole", requestBody).Return(int64(0), internalErr)

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, internalErr.Error(), response.Message)

	mockService.AssertExpectations(t)
}

func TestController_CreateRole_WithParentId(t *testing.T) {
	app, mockService := setupTestApp()

	parentId := int64(456)
	requestBody := CreateRequest{
		Name:        "Child Role",
		Description: "Child Role Description",
		Status:      true,
		ParentId:    &parentId,
	}

	expectedRoleId := int64(789)

	mockService.On("CreateRole", requestBody).Return(expectedRoleId, nil)

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response struct {
		Success bool           `json:"success"`
		Message string         `json:"error"`
		Data    map[string]any `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	id, ok := response.Data["id"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(expectedRoleId), id)

	mockService.AssertExpectations(t)
}

func TestController_CreateRole_EmptyBody(t *testing.T) {
	// Arrange
	app, mockService := setupTestApp()

	// пустой body
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer([]byte("")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	mockService.AssertNotCalled(t, "CreateRole")
}

func TestController_CreateRole_MissingContentType(t *testing.T) {
	app, mockService := setupTestApp()

	requestBody := CreateRequest{
		Name:        "Test Role",
		Description: "Test Description",
		Status:      true,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(jsonBody))
	// Не устанавливаем Content-Type

	resp, err := app.Test(req)

	assert.NoError(t, err)
	// Fiber не может корректно распарсить JSON без Content-Type
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotEmpty(t, response.Message)

	// Mock не должен был вызываться из-за ошибки парсинга
	mockService.AssertNotCalled(t, "CreateRole")
}

func TestController_CreateRole_InvalidData_ReturnsValidationError(t *testing.T) {
	app, mockService := setupTestApp()

	parentId := int64(56)

	// Невалидные данные (пустое имя)
	createRequest := CreateRequest{
		Name:        "",
		Description: "Some description",
		Status:      true,
		ParentId:    &parentId,
	}
	validationError := common.RequestValidationError{Message: "validation failed"}
	mockService.On("CreateRole", createRequest).Return(int64(0), validationError)

	requestBody, _ := json.Marshal(createRequest)
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "validation failed", response.Message)

	mockService.AssertExpectations(t)
}
