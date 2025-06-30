package employee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"idm/inner/common"
	"idm/inner/web"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) FindById(ctx context.Context, id int64) (Response, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Response), args.Error(1)
}

func (m *MockService) CreateEmployee(ctx context.Context, request CreateRequest) (int64, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockService) DeleteById(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockService) DeleteByIds(ctx context.Context, ids []int64) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockService) FindAll(ctx context.Context) ([]Response, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Response), args.Error(1)
}

func (m *MockService) FindByIds(ctx context.Context, ids []int64) ([]Response, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]Response), args.Error(1)
}

func (m *MockService) FindWithPagination(ctx context.Context, request PageRequest) (PageResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(PageResponse), args.Error(1)
}

// setupTestController - вспомогательная функция для создания тестового контроллера
func setupTestController(t *testing.T) (*MockService, *fiber.App) {
	app := fiber.New()

	server := &web.Server{
		App:        app,
		GroupApiV1: app.Group("/api/v1"),
	}

	mockService := &MockService{}

	cfg := common.Config{
		DbDriverName:   "postgres",
		Dsn:            "localhost port=5432 user=wronguser password=wrongpass dbname=postgres sslmode=disable",
		AppName:        "test_app",
		AppVersion:     "1.0.0",
		LogLevel:       "DEBUG",
		LogDevelopMode: true,
		SslSert:        "ssl.cert",
		SslKey:         "ssl.key",
	}

	logger := common.NewLogger(cfg)

	controller := NewController(server, mockService, logger)

	controller.RegisterRoutes()

	// Очистка переменных окружения после теста
	t.Cleanup(func() {
		_ = os.Unsetenv("DB_DRIVER_NAME")
		_ = os.Unsetenv("DB_DSN")
		_ = os.Unsetenv("APP_NAME")
		_ = os.Unsetenv("APP_VERSION")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("LOG_DEVELOP_MODE")
		_ = os.Unsetenv("SSL_SERT")
		_ = os.Unsetenv("SSL_KEY")
	})

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
	mockService.On("CreateEmployee", mock.Anything, createRequest).Return(expectedEmployeeId, nil)

	requestBody, _ := json.Marshal(createRequest)
	req := httptest.NewRequest("POST", "/api/v1/employees", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

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
	assert.Equal(t, float64(expectedEmployeeId), id)

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
	mockService.On("CreateEmployee", mock.Anything, createRequest).Return(int64(0), validationError)

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
	mockService.On("CreateEmployee", mock.Anything, createRequest).Return(int64(0), alreadyExistsError)

	requestBody, _ := json.Marshal(createRequest)
	req := httptest.NewRequest("POST", "/api/v1/employees", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

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

	internalError := errors.New("Internal server error")
	mockService.On("CreateEmployee", mock.Anything, createRequest).Return(int64(0), internalError)

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
	assert.Equal(t, "Internal server error", response.Message)

	mockService.AssertExpectations(t)
}

func TestController_CreateEmployee_InvalidData_ReturnsValidationError(t *testing.T) {
	mockService, app := setupTestController(t)

	// Невалидные данные (пустое имя)
	createRequest := CreateRequest{
		Name:       "",
		Email:      "test@example.com",
		Position:   "Dev",
		Department: "IT",
		RoleId:     1,
	}
	validationError := common.RequestValidationError{Message: "validation failed"}
	mockService.On("CreateEmployee", mock.Anything, createRequest).Return(int64(0), validationError)

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

// тестирует валидацию параметров пагинации
func TestFindEmployeesWithPagination_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		pageNumber     string
		pageSize       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "PageSize less than 1",
			pageNumber:     "1",
			pageSize:       "0",
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Error when getting paginated employees",
		},
		{
			name:           "PageSize negative",
			pageNumber:     "1",
			pageSize:       "-5",
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Error when getting paginated employees",
		},
		{
			name:           "PageSize greater than 100",
			pageNumber:     "1",
			pageSize:       "101",
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Error when getting paginated employees",
		},
		{
			name:           "PageNumber less than 1 (zero)",
			pageNumber:     "0",
			pageSize:       "10",
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Error when getting paginated employees",
		},
		{
			name:           "PageNumber negative",
			pageNumber:     "-1",
			pageSize:       "10",
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Error when getting paginated employees",
		},
		{
			name:           "Invalid pageNumber format",
			pageNumber:     "abc",
			pageSize:       "10",
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Invalid pageNumber parameter",
		},
		{
			name:           "Invalid pageSize format",
			pageNumber:     "1",
			pageSize:       "xyz",
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Invalid pageSize parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService, app := setupTestController(t)

			mockService.On("FindWithPagination", mock.Anything, mock.AnythingOfType("PageRequest")).
				Return(PageResponse{}, errors.New("validation error")).Once()

			url := fmt.Sprintf("/api/v1/employees/page?pageNumber=%s&pageSize=%s", tt.pageNumber, tt.pageSize)

			req := httptest.NewRequest("GET", url, nil)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Проверка тела ответа
			var responseBody common.Response[any]
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			require.NoError(t, err)

			assert.Contains(t, responseBody.Message, tt.expectedError)
			assert.False(t, responseBody.Success)
		})
	}
}

// Тестирует успешный сценарий
func TestFindEmployeesWithPagination_Success(t *testing.T) {
	mockService, app := setupTestController(t)

	expectedResponse := PageResponse{
		Data: []Response{
			{Id: 4, Name: "Rick Sanchez", Email: "rick@example.com", Position: "Developer", Department: "Engineering", RoleId: 3, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{Id: 2, Name: "Jane Smith", Email: "jane@example.com", Position: "Developer", Department: "IT", RoleId: 3, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{Id: 5, Name: "John Doe", Email: "john@example.com", Position: "CTO", Department: "IT", RoleId: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
		PageNumber: 1,
		PageSize:   10,
		TotalCount: 3,
		TotalPages: 1,
	}

	expectedRequest := PageRequest{
		PageNumber: 1,
		PageSize:   10,
	}

	mockService.On("FindWithPagination", mock.Anything, expectedRequest).Return(expectedResponse, nil).Once()

	req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=1&pageSize=10", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	if resp.StatusCode != fiber.StatusOK {
		var errorBody common.Response[any]
		err = json.NewDecoder(resp.Body).Decode(&errorBody)
		require.NoError(t, err)
		t.Logf("Error response: %+v", errorBody)
	}

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var responseBody common.Response[PageResponse]
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	require.NoError(t, err)

	assert.True(t, responseBody.Success)
	assert.NotNil(t, responseBody.Data)

	data := responseBody.Data
	assert.Equal(t, 1, data.PageNumber)
	assert.Equal(t, 10, data.PageSize)
	assert.Equal(t, int64(3), data.TotalCount)
	assert.Equal(t, 1, data.TotalPages)

	assert.Len(t, data.Data, 3)

	mockService.AssertExpectations(t)
}

// Тестирует использование значений по умолчанию
func TestFindEmployeesWithPagination_DefaultValues(t *testing.T) {
	mockService, app := setupTestController(t)

	expectedResponse := PageResponse{
		Data:       []Response{},
		PageNumber: 1,
		PageSize:   10,
		TotalCount: 0,
		TotalPages: 1,
	}

	// значения по умолчанию
	expectedRequest := PageRequest{
		PageNumber: 1,
		PageSize:   10,
	}

	mockService.On("FindWithPagination", mock.Anything, expectedRequest).
		Return(expectedResponse, nil).
		Once()

	// HTTP запрос без параметров (значения по умолчанию)
	req := httptest.NewRequest("GET", "/api/v1/employees/page", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var responseBody common.Response[PageResponse]
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	require.NoError(t, err)

	assert.True(t, responseBody.Success)

	data := responseBody.Data
	assert.Equal(t, 1, data.PageNumber)
	assert.Equal(t, 10, data.PageSize)

	mockService.AssertExpectations(t)
}

// Тестирует обработку ошибок сервиса
func TestFindEmployeesWithPagination_ServiceError(t *testing.T) {
	mockService, app := setupTestController(t)

	mockService.On("FindWithPagination", mock.Anything, mock.Anything).
		Return(PageResponse{}, errors.New("Invalid pagination request")).
		Once()

	req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=1&pageSize=10", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var responseBody common.Response[any]
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	require.NoError(t, err)

	assert.Contains(t, responseBody.Message, "Error when getting paginated employees")
	assert.False(t, responseBody.Success)

	mockService.AssertExpectations(t)
}

func TestNewController(t *testing.T) {
	app := fiber.New()
	server := &web.Server{
		App:        app,
		GroupApiV1: app.Group("/api/v1"),
	}
	mockService := &MockService{}

	// Создаем временную директорию
	tempDir := t.TempDir()

	// Путь к тестовому .env файлу
	envFilePath := filepath.Join(tempDir, ".env")

	// Записываем в него данные
	dotEnvContent := []byte(`
	DB_DRIVER_NAME=postgres
	DB_DSN=host=localhost port=5432 user=wronguser password=wrongpass dbname=postgres sslmode=disable
	APP_NAME=test-app
	APP_VERSION=1.0.0
	LOG_LEVEL=DEBUG
	LOG_DEVELOP_MODE=true
	SSL_SERT=ssl.cert
	SSL_KEY=ssl.key
	`)
	err := os.WriteFile(envFilePath, dotEnvContent, 0644)
	require.NoError(t, err)

	// Получаем конфиг из файла
	cfg := common.GetConfig(envFilePath)
	assert.NotEmpty(t, cfg)

	logger := common.NewLogger(cfg)

	controller := NewController(server, mockService, logger)

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
