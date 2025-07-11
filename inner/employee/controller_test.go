//go:build integration
// +build integration

package employee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"fmt"
	"idm/inner/common"
	"idm/inner/testutils"
	"idm/inner/web"
	"io"
	"net/http"
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

// setupTestServer создает тестовый сервер с настроенной аутентификацией
func setupTestServer(t *testing.T) (*MockService, *fiber.App) {

	cfg := common.Config{
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

	logger := common.NewLogger(cfg)

	server := web.NewServer(logger)

	mockService := &MockService{}
	controller := NewController(server, mockService, logger)
	controller.RegisterRoutes()

	// Очистка после теста
	t.Cleanup(func() {
		os.Clearenv()
	})

	return mockService, server.App
}

// HTTP запрос с валидным JWT токеном
func createAuthenticatedRequest(t *testing.T, method, url string, body io.Reader, userRoles []string) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")

	if userRoles == nil {
		return req // без токена
	}

	cfg, _ := testutils.LoadTestConfig("..", "")

	// Мапа для соответствия ролей и пользователей
	roleToUserMap := map[string]struct {
		Username string
		Password string
	}{
		web.IdmAdmin: {Username: cfg.Keycloak.Username1, Password: cfg.Keycloak.Password},
		web.IdmUser:  {Username: cfg.Keycloak.Username2, Password: cfg.Keycloak.Password},
	}

	// Дефолтные данные
	username := "testuser"
	password := "password"

	// Выбираем пользователя по первой роли
	if len(userRoles) > 0 {
		if creds, ok := roleToUserMap[userRoles[0]]; ok {
			username = creds.Username
			password = creds.Password
		}

		// Получаем реальный токен из Keycloak
		ctx := context.Background()
		accessToken, err := testutils.GetKeycloakToken(
			ctx,
			cfg.Keycloak.Realm,        // realm
			cfg.Keycloak.ClientID,     // client ID
			cfg.Keycloak.ClientSecret, // client secret (из Keycloak)
			username,                  // username
			password,                  // password
		)
		require.NoError(t, err)
		require.NotEmpty(t, accessToken)

		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	return req
}

// создаёт запрос без заголовка Authorization
func createUnauthenticatedRequest(method, url string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// создаёт запрос с испорченным токеном
func createMalformedTokenRequest(method, url string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid_token_here_123")
	return req
}

// создаёт запрос с просроченным токеном
func createExpiredTokenRequest(method, url string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutils.GenerateExpiredToken())
	return req
}

// запрос с токеном, у которого нет нужной роли
func createForbiddenTokenRequest(method, url string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutils.GenerateRolelessToken())
	return req
}

func TestController_CreateEmployee(t *testing.T) {
	tests := []struct {
		name         string
		userRoles    []string
		requestBody  CreateRequest
		mockSetup    func(*MockService)
		expectedCode int
		expectedData any
		expectError  bool
	}{
		{
			name:        "successful creation with admin role",
			userRoles:   []string{web.IdmAdmin},
			requestBody: CreateRequest{Name: "John Doe", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 1},
			mockSetup: func(m *MockService) {
				m.On("CreateEmployee", mock.Anything, mock.AnythingOfType("CreateRequest")).
					Return(int64(123), nil)
			},
			expectedCode: http.StatusOK,
			expectedData: float64(123),
			expectError:  false,
		},
		{
			name:        "forbidden access with user role",
			userRoles:   []string{web.IdmUser},
			requestBody: CreateRequest{Name: "John Doe", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 2},
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться при отсутствии прав
			},
			expectedCode: http.StatusForbidden,
			expectError:  true,
		},
		{
			name:        "validation error",
			userRoles:   []string{web.IdmAdmin},
			requestBody: CreateRequest{Name: "", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 2}, // пустое имя
			mockSetup: func(m *MockService) {
				validationErr := common.RequestValidationError{
					Message: "Name is required",
					Data:    map[string]any{"name": "required"},
				}
				m.On("CreateEmployee", mock.Anything, mock.AnythingOfType("CreateRequest")).
					Return(int64(0), validationErr)
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		{
			name:        "already exists error",
			userRoles:   []string{web.IdmAdmin},
			requestBody: CreateRequest{Name: "John Doe", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 2},
			mockSetup: func(m *MockService) {
				existsErr := common.AlreadyExistsError{Message: "Employee already exists"}
				m.On("CreateEmployee", mock.Anything, mock.AnythingOfType("CreateRequest")).
					Return(int64(0), existsErr)
			},
			expectedCode: http.StatusConflict,
			expectError:  true,
		},
		{
			name:        "internal server error",
			userRoles:   []string{web.IdmAdmin},
			requestBody: CreateRequest{Name: "John Doe", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 2},
			mockSetup: func(m *MockService) {
				m.On("CreateEmployee", mock.Anything, mock.AnythingOfType("CreateRequest")).
					Return(int64(0), errors.New("database connection error"))
			},
			expectedCode: http.StatusInternalServerError,
			expectError:  true,
		},
		{
			name:        "unauthorized missing token",
			userRoles:   nil,
			requestBody: CreateRequest{Name: "John Doe", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 1},
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:        "unauthorized malformed token",
			userRoles:   nil,
			requestBody: CreateRequest{Name: "John Doe", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 1},
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:        "unauthorized malformed token",
			userRoles:   []string{web.IdmAdmin},
			requestBody: CreateRequest{Name: "John Doe", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 1},
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:        "unauthorized expired token",
			userRoles:   []string{web.IdmAdmin},
			requestBody: CreateRequest{Name: "John Doe", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 1},
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:        "forbidden access with user role",
			userRoles:   []string{web.IdmUser}, // нет прав на /admin
			requestBody: CreateRequest{Name: "John Doe", Email: "john@example.com", Position: "Developer", Department: "IT", RoleId: 2},
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusForbidden,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService, app := setupTestServer(t)

			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			requestBody, _ := json.Marshal(tt.requestBody)

			var req *http.Request
			switch {
			case tt.userRoles == nil:
				req = createUnauthenticatedRequest(
					fiber.MethodPost,
					"/api/v1/admin/employees",
					bytes.NewReader(requestBody),
				)

			case len(tt.userRoles) > 0 && tt.name == "unauthorized malformed token":
				// токен невалидный
				req = createMalformedTokenRequest(
					fiber.MethodPost,
					"/api/v1/admin/employees",
					bytes.NewReader(requestBody),
				)

			case len(tt.userRoles) > 0 && tt.name == "unauthorized expired token":
				// токен истёк по времени
				req = createExpiredTokenRequest(
					fiber.MethodPost,
					"/api/v1/admin/employees",
					bytes.NewReader(requestBody),
				)

			default:
				req = createAuthenticatedRequest(
					t,
					fiber.MethodPost,
					"/api/v1/admin/employees",
					bytes.NewReader(requestBody),
					tt.userRoles,
				)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var responseBody common.Response[any]
			err = json.Unmarshal(bodyBytes, &responseBody)
			require.NoError(t, err)

			if tt.expectError {
				assert.False(t, responseBody.Success)
				assert.NotEmpty(t, responseBody.Message)
			} else {
				assert.True(t, responseBody.Success)
				if tt.expectedData != nil {
					if data, ok := responseBody.Data.(map[string]any); ok {
						assert.Equal(t, tt.expectedData, data["id"])
					}
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestController_GetEmployee(t *testing.T) {
	tests := []struct {
		name         string
		userRoles    []string
		employeeId   string
		requestFunc  func(method, url string, body io.Reader) *http.Request
		mockSetup    func(*MockService)
		expectedCode int
		expectError  bool
	}{
		{
			name:       "successful get with admin role",
			userRoles:  []string{web.IdmAdmin},
			employeeId: "123",
			requestFunc: func(method, url string, body io.Reader) *http.Request {
				return createAuthenticatedRequest(t, method, url, body, []string{web.IdmAdmin})
			},
			mockSetup: func(m *MockService) {
				response := Response{
					Id:   123,
					Name: "John Doe",
				}
				m.On("FindById", mock.Anything, int64(123)).
					Return(response, nil)
			},
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		{
			name:       "successful get with user role",
			userRoles:  []string{web.IdmUser},
			employeeId: "123",
			requestFunc: func(method, url string, body io.Reader) *http.Request {
				return createAuthenticatedRequest(t, method, url, body, []string{web.IdmUser})
			},
			mockSetup: func(m *MockService) {
				response := Response{
					Id:   123,
					Name: "Marry Beth",
				}
				m.On("FindById", mock.Anything, int64(123)).
					Return(response, nil)
			},
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		{
			name:        "unauthorized missing token",
			userRoles:   nil,
			employeeId:  "123",
			requestFunc: createUnauthenticatedRequest,
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:        "unauthorized malformed token",
			userRoles:   nil,
			employeeId:  "123",
			requestFunc: createMalformedTokenRequest,
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:        "unauthorized expired token",
			userRoles:   nil,
			employeeId:  "123",
			requestFunc: createExpiredTokenRequest,
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:        "forbidden access without required role",
			userRoles:   nil,
			employeeId:  "123",
			requestFunc: createForbiddenTokenRequest,
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться при отсутствии нужных ролей
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:       "not found error",
			userRoles:  []string{web.IdmAdmin},
			employeeId: "999",
			requestFunc: func(method, url string, body io.Reader) *http.Request {
				return createAuthenticatedRequest(t, method, url, body, []string{web.IdmAdmin})
			},
			mockSetup: func(m *MockService) {
				notFoundErr := common.NotFoundError{Message: "Employee not found"}
				m.On("FindById", mock.Anything, int64(999)).
					Return(Response{}, notFoundErr)
			},
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		{
			name:       "invalid employee id",
			userRoles:  []string{web.IdmAdmin},
			employeeId: "invalid",
			requestFunc: func(method, url string, body io.Reader) *http.Request {
				return createAuthenticatedRequest(t, method, url, body, []string{web.IdmAdmin})
			},
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться при невалидном ID
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService, app := setupTestServer(t)

			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			url := "/api/v1/employees/" + tt.employeeId
			req := tt.requestFunc(fiber.MethodGet, url, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var responseBody common.Response[any]
			err = json.Unmarshal(bodyBytes, &responseBody)
			require.NoError(t, err)

			if tt.expectError {
				assert.False(t, responseBody.Success)
				assert.NotEmpty(t, responseBody.Message)
			} else {
				assert.True(t, responseBody.Success)
				assert.NotNil(t, responseBody.Data)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestController_DeleteEmployee(t *testing.T) {
	tests := []struct {
		name         string
		userRoles    []string
		employeeId   string
		requestFunc  func(method, url string, body io.Reader) *http.Request
		mockSetup    func(*MockService)
		expectedCode int
		expectError  bool
	}{
		{
			name:       "successful delete with admin role",
			userRoles:  []string{web.IdmAdmin},
			employeeId: "123",
			requestFunc: func(method, url string, body io.Reader) *http.Request {
				return createAuthenticatedRequest(t, method, url, body, []string{web.IdmAdmin})
			},
			mockSetup: func(m *MockService) {
				m.On("DeleteById", mock.Anything, int64(123)).
					Return(nil)
			},
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		{
			name:       "forbidden access with user role",
			userRoles:  []string{web.IdmUser},
			employeeId: "123",
			requestFunc: func(method, url string, body io.Reader) *http.Request {
				return createAuthenticatedRequest(t, method, url, body, []string{web.IdmUser})
			},
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться при отсутствии прав
			},
			expectedCode: http.StatusForbidden,
			expectError:  true,
		},
		{
			name:        "unauthorized missing token",
			userRoles:   nil,
			employeeId:  "123",
			requestFunc: createUnauthenticatedRequest,
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:        "unauthorized malformed token",
			userRoles:   nil,
			employeeId:  "123",
			requestFunc: createMalformedTokenRequest,
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:        "unauthorized expired token",
			userRoles:   nil,
			employeeId:  "123",
			requestFunc: createExpiredTokenRequest,
			mockSetup: func(m *MockService) {
				// Сервис не должен вызываться
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name:       "not found error",
			userRoles:  []string{web.IdmAdmin},
			employeeId: "999",
			requestFunc: func(method, url string, body io.Reader) *http.Request {
				return createAuthenticatedRequest(t, method, url, body, []string{web.IdmAdmin})
			},
			mockSetup: func(m *MockService) {
				notFoundErr := common.NotFoundError{Message: "Employee not found"}
				m.On("DeleteById", mock.Anything, int64(999)).
					Return(notFoundErr)
			},
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService, app := setupTestServer(t)

			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			url := "/api/v1/admin/employees/" + tt.employeeId
			req := tt.requestFunc(fiber.MethodDelete, url, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var responseBody common.Response[any]
			err = json.Unmarshal(bodyBytes, &responseBody)
			require.NoError(t, err)

			if tt.expectError {
				assert.False(t, responseBody.Success)
				assert.NotEmpty(t, responseBody.Message)
			} else {
				assert.True(t, responseBody.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestController_UnauthorizedAccess(t *testing.T) {
	// Создаем сервер БЕЗ middleware (имитируем реальную ситуацию без аутентификации)
	cfg := common.Config{
		DbDriverName:   "postgres",
		Dsn:            "test_dsn",
		AppName:        "test_app",
		AppVersion:     "1.0.0",
		LogLevel:       "DEBUG",
		LogDevelopMode: true,
		SslSert:        "ssl.cert",
		SslKey:         "ssl.key",
		KeycloakJwkUrl: "http://localhost:9990/realms/idm/protocol/openid-connect/certs",
	}

	logger := common.NewLogger(cfg)
	// Используем настоящий сервер с JWT middleware для тестирования неавторизованного доступа
	server := web.NewServer(logger)

	mockService := &MockService{}
	controller := NewController(server, mockService, logger)
	controller.RegisterRoutes()

	tests := []struct {
		name   string
		method string
		url    string
		body   io.Reader
	}{
		{
			name:   "create employee without auth",
			method: fiber.MethodPost,
			url:    "/api/v1/employees",
			body:   strings.NewReader(`{"name": "John Doe"}`),
		},
		{
			name:   "get employee without auth",
			method: fiber.MethodGet,
			url:    "/api/v1/employees/123",
			body:   nil,
		},
		{
			name:   "delete employee without auth",
			method: fiber.MethodDelete,
			url:    "/api/v1/employees/123",
			body:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, tt.body)
			req.Header.Set("Content-Type", "application/json")
			// Намеренно НЕ добавляем Authorization header

			resp, err := server.App.Test(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			// Без аутентификации должен возвращаться 401
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestController_CreateEmployee_Success(t *testing.T) {
	mockService, app := setupTestServer(t)

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
	req := createAuthenticatedRequest(t, fiber.MethodPost, "/api/v1/admin/employees", bytes.NewReader(requestBody), []string{web.IdmAdmin})

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
	_, app := setupTestServer(t)

	// Подготавливаем некорректный JSON
	req := createAuthenticatedRequest(t, fiber.MethodPost, "/api/v1/admin/employees", bytes.NewReader([]byte("invalid json")), []string{web.IdmAdmin})

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
	mockService, app := setupTestServer(t)

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
	req := createAuthenticatedRequest(t, fiber.MethodPost, "/api/v1/admin/employees", bytes.NewReader(requestBody), []string{web.IdmAdmin})

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
	mockService, app := setupTestServer(t)

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
	req := createAuthenticatedRequest(t, fiber.MethodPost, "/api/v1/admin/employees", bytes.NewReader(requestBody), []string{web.IdmAdmin})

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

	mockService, app := setupTestServer(t)

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
	req := createAuthenticatedRequest(t, fiber.MethodPost, "/api/v1/admin/employees", bytes.NewReader(requestBody), []string{web.IdmAdmin})

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
	mockService, app := setupTestServer(t)

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
	req := createAuthenticatedRequest(t, fiber.MethodPost, "/api/v1/admin/employees", bytes.NewReader(requestBody), []string{web.IdmAdmin})

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
			mockService, app := setupTestServer(t)

			mockService.On("FindWithPagination", mock.Anything, mock.AnythingOfType("PageRequest")).
				Return(PageResponse{}, errors.New("validation error")).Once()

			url := fmt.Sprintf("/api/v1/employees/page?pageNumber=%s&pageSize=%s", tt.pageNumber, tt.pageSize)

			req := createAuthenticatedRequest(t, fiber.MethodGet, url, nil, []string{web.IdmUser})

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
	mockService, app := setupTestServer(t)

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

	req := createAuthenticatedRequest(t, fiber.MethodGet, "/api/v1/employees/page?pageNumber=1&pageSize=10", nil, []string{web.IdmUser})

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
	mockService, app := setupTestServer(t)

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
	req := createAuthenticatedRequest(t, fiber.MethodGet, "/api/v1/employees/page", nil, []string{web.IdmUser})

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
	mockService, app := setupTestServer(t)

	mockService.On("FindWithPagination", mock.Anything, mock.Anything).
		Return(PageResponse{}, errors.New("Invalid pagination request")).
		Once()

	req := createAuthenticatedRequest(t, fiber.MethodGet, "/api/v1/employees/page?pageNumber=1&pageSize=10", nil, []string{web.IdmUser})

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
	KEYCLOAK_JWK_URL=http://localhost:9990/realms/idm/protocol/openid-connect/certs 
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
	_, app := setupTestServer(t)

	// Проверка, что маршрут был зарегистрирован
	// Тестовый запрос с некорректными данными
	req := createAuthenticatedRequest(t, fiber.MethodPost, "/api/v1/employees", bytes.NewReader([]byte("test")), []string{web.IdmAdmin})

	resp, err := app.Test(req)

	// Если маршрут зарегистрирован, получим ответ (не 404)
	assert.NoError(t, err)
	assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
}
