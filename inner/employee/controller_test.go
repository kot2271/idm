package employee

import (
	"bytes"
	"encoding/json"
	"errors"
	"idm/inner/common"
	"idm/inner/web"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) FindById(id int64) (Response, error) {
	args := m.Called(id)
	return args.Get(0).(Response), args.Error(1)
}

func (m *MockService) CreateEmployee(request CreateRequest) (int64, error) {
	args := m.Called(request)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockService) DeleteById(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockService) DeleteByIds(ids []int64) error {
	args := m.Called(ids)
	return args.Error(0)
}

func (m *MockService) FindAll() ([]Response, error) {
	args := m.Called()
	return args.Get(0).([]Response), args.Error(1)
}

func (m *MockService) FindByIds(ids []int64) ([]Response, error) {
	args := m.Called(ids)
	return args.Get(0).([]Response), args.Error(1)
}

// setupTestController - вспомогательная функция для создания тестового контроллера
func setupTestController(_ *testing.T) (*MockService, *fiber.App) {
	app := fiber.New()

	server := &web.Server{
		App:        app,
		GroupApiV1: app.Group("/api/v1"),
	}

	mockService := &MockService{}

	controller := NewController(server, mockService)

	controller.RegisterRoutes()

	return mockService, app
}

func TestController_CreateEmployee_Success(t *testing.T) {
	mockService, app := setupTestController(t)

	createRequest := CreateRequest{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     1,
	}

	expectedEmployeeId := int64(123)
	mockService.On("CreateEmployee", createRequest).Return(expectedEmployeeId, nil)

	requestBody, _ := json.Marshal(createRequest)
	req := httptest.NewRequest("POST", "/api/v1/employees", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response common.Response[int64]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, expectedEmployeeId, response.Data)

	mockService.AssertExpectations(t)
}

func TestController_CreateEmployee_InvalidJSON(t *testing.T) {
	_, app := setupTestController(t)

	// Подготавливаем некорректный JSON
	req := httptest.NewRequest("POST", "/api/v1/employees", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotEmpty(t, response.Message)
}

func TestController_CreateEmployee_ValidationError(t *testing.T) {
	mockService, app := setupTestController(t)

	createRequest := CreateRequest{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     1,
	}

	validationError := common.RequestValidationError{Message: "validation failed"}
	mockService.On("CreateEmployee", createRequest).Return(int64(0), validationError)

	requestBody, _ := json.Marshal(createRequest)
	req := httptest.NewRequest("POST", "/api/v1/employees", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "validation failed", response.Message)

	mockService.AssertExpectations(t)
}

func TestController_CreateEmployee_AlreadyExistsError(t *testing.T) {
	mockService, app := setupTestController(t)

	createRequest := CreateRequest{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     1,
	}

	alreadyExistsError := common.AlreadyExistsError{Message: "employee already exists"}
	mockService.On("CreateEmployee", createRequest).Return(int64(0), alreadyExistsError)

	requestBody, _ := json.Marshal(createRequest)
	req := httptest.NewRequest("POST", "/api/v1/employees", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "employee already exists", response.Message)

	mockService.AssertExpectations(t)
}

func TestController_CreateEmployee_InternalServerError(t *testing.T) {

	mockService, app := setupTestController(t)

	createRequest := CreateRequest{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     1,
	}

	internalError := errors.New("database connection failed")
	mockService.On("CreateEmployee", createRequest).Return(int64(0), internalError)

	requestBody, _ := json.Marshal(createRequest)
	req := httptest.NewRequest("POST", "/api/v1/employees", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var response common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "database connection failed", response.Message)

	mockService.AssertExpectations(t)
}

func TestNewController(t *testing.T) {
	app := fiber.New()
	server := &web.Server{
		App:        app,
		GroupApiV1: app.Group("/api/v1"),
	}
	mockService := &MockService{}

	controller := NewController(server, mockService)

	assert.NotNil(t, controller)
	assert.Equal(t, server, controller.server)
	assert.Equal(t, mockService, controller.employeeService)
}

func TestController_RegisterRoutes(t *testing.T) {
	_, app := setupTestController(t)

	// Проверка, что маршрут был зарегистрирован
	// Тестовый запрос с некорректными данными
	req := httptest.NewRequest("POST", "/api/v1/employees", bytes.NewReader([]byte("test")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	// Если маршрут зарегистрирован, получим ответ (не 404)
	assert.NoError(t, err)
	assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
}
