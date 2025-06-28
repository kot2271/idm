package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"idm/inner/common"
	"idm/inner/employee"
	val "idm/inner/validator"
	"idm/inner/web"

	"github.com/gofiber/fiber/v2"
	"github.com/icrowley/fake"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func appLaunchKit() *fiber.App {
	server := web.NewServer()
	validator := val.New()
	logger := common.NewLogger(config)

	repo := employee.NewEmployeeRepository(DB)
	service := employee.NewService(repo, validator, logger)
	controller := employee.NewController(server, service, logger)

	app := server.App

	api := server.GroupApiV1
	api.Get("/employees/page", controller.FindEmployeesWithPagination)

	return app
}

func TestEmployeeRepository_CRUD(t *testing.T) {
	repo := employee.NewEmployeeRepository(DB)

	clearTables()

	// Добавляем роль
	var roleID int64 = 1
	err := DB.QueryRow(`INSERT INTO role (name) VALUES ($1) RETURNING id`, "Test Role").Scan(&roleID)
	assert.NoError(t, err)

	// Создаем сотрудника
	emp := &employee.Entity{
		Name:       "John Doe",
		Email:      "john@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     roleID,
	}

	emp2 := &employee.Entity{
		Name:       "Rick Sanchez",
		Email:      "rick@example.com",
		Position:   "Manager",
		Department: "IT",
		RoleId:     roleID,
	}

	t.Run("Add", func(t *testing.T) {
		err := repo.Add(context.Background(), emp)
		assert.NoError(t, err)
		assert.NotZero(t, emp.Id)

		err = repo.Add(context.Background(), emp2)
		assert.NoError(t, err)
		assert.NotZero(t, emp2.Id)
	})

	t.Run("FindById", func(t *testing.T) {
		found, err := repo.FindById(context.Background(), emp.Id)
		assert.NoError(t, err)
		assert.Equal(t, emp.Name, found.Name)
		assert.Equal(t, emp.Email, found.Email)
	})

	t.Run("FindAll", func(t *testing.T) {
		employees, err := repo.FindAll(context.Background())
		assert.NoError(t, err)
		assert.Len(t, employees, 2)
		assert.Equal(t, emp.Email, employees[0].Email)
		assert.Equal(t, emp2.Email, employees[1].Email)
	})

	t.Run("FindByIds", func(t *testing.T) {
		employees, err := repo.FindByIds(context.Background(), []int64{emp.Id, emp2.Id})
		assert.NoError(t, err)
		assert.Len(t, employees, 2)
		assert.Equal(t, emp.Id, employees[0].Id)
		assert.Equal(t, emp2.Id, employees[1].Id)
	})

	t.Run("DeleteById", func(t *testing.T) {
		err := repo.DeleteById(context.Background(), emp.Id)
		assert.NoError(t, err)

		_, err = repo.FindById(context.Background(), emp.Id)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rows in result set")
	})

	t.Run("DeleteByIds", func(t *testing.T) {
		// Добавляем двух сотрудников
		e1 := &employee.Entity{Name: "Alice", Email: "alice@example.com", RoleId: roleID}
		e2 := &employee.Entity{Name: "Bob", Email: "bob@example.com", RoleId: roleID}
		_ = repo.Add(context.Background(), e1)
		_ = repo.Add(context.Background(), e2)

		err := repo.DeleteByIds(context.Background(), []int64{e1.Id, e2.Id})
		assert.NoError(t, err)

		_, err = repo.FindById(context.Background(), e1.Id)
		assert.Error(t, err)
		_, err = repo.FindById(context.Background(), e2.Id)
		assert.Error(t, err)
	})
}

func TestAddWithTransaction_Success(t *testing.T) {
	repo := employee.NewEmployeeRepository(DB)

	clearTables()

	tx, err := repo.BeginTransaction(context.Background())
	if err != nil {
		t.Fatalf("Error beginning transaction: %v", err)
	}

	var roleID int64 = 1
	err = DB.QueryRow(`INSERT INTO role (name) VALUES ($1) RETURNING id`, "Test Role").Scan(&roleID)
	assert.NoError(t, err)

	empl := &employee.Entity{
		Name:       "John Doe",
		Email:      "john@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     roleID,
	}

	// Execute AddWithTransaction
	err = repo.AddWithTransaction(context.Background(), tx, empl)
	if err != nil {
		t.Fatalf("Error adding with transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Error committing: %v", err)
	}

	// Verify that the employee exists in the database
	var result employee.Entity
	err = DB.Get(&result, "SELECT * FROM employee WHERE name = $1", empl.Name)
	if err != nil {
		t.Fatalf("Failed to query employee: %v", err)
	}

	if result.Name != empl.Name {
		t.Errorf("Expected name %s, got %s", empl.Name, result.Name)
	}
}

func TestAddWithTransaction_Failure(t *testing.T) {
	repo := employee.NewEmployeeRepository(DB)
	clearTables()

	// Begin transaction
	tx, err := repo.BeginTransaction(context.Background())
	if err != nil {
		t.Fatalf("Error beginning transaction: %v", err)
	}

	// Insert role inside test
	var roleID int64
	err = DB.QueryRow(`INSERT INTO role (name) VALUES ($1) RETURNING id`, "Test Role").Scan(&roleID)
	assert.NoError(t, err)

	// Insert original employee inside the transaction
	existingEmp := &employee.Entity{
		Name:       "Jane Smith",
		Email:      "jane@example.com",
		Position:   "Manager",
		Department: "HR",
		RoleId:     roleID,
	}

	_, err = tx.Exec("INSERT INTO employee (name, email, position, department, role_id) VALUES ($1, $2, $3, $4, $5)",
		existingEmp.Name, existingEmp.Email, existingEmp.Position, existingEmp.Department, existingEmp.RoleId)
	if err != nil {
		t.Fatalf("Failed to insert existing employee in transaction: %v", err)
	}

	// Try to insert duplicate employee inside the same transaction
	duplicateEmp := &employee.Entity{
		Name:       "Jane Smith", // duplicate
		Email:      "jane2@example.com",
		Position:   "Sale_Manager",
		Department: "Sales",
		RoleId:     3,
	}

	err = repo.AddWithTransaction(context.Background(), tx, duplicateEmp)
	if err == nil {
		t.Fatalf("Expected error on duplicate name, but got nil")
	}

	// Rollback transaction — all changes should be undone
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Error rolling back transaction: %v", err)
	}

	var result employee.Entity

	err = DB.Get(&result, "SELECT * FROM employee WHERE name = $1", existingEmp.Name)
	if err == nil {
		t.Errorf("Expected no employee with name %s, but found one after rollback", existingEmp.Name)
	}

	err = DB.Get(&result, "SELECT * FROM employee WHERE name = $1", duplicateEmp.Name)
	if err == nil {
		t.Errorf("Expected no employee with name %s, but found one after rollback", duplicateEmp.Name)
	}
}

func TestBeginTransactionEmployee(t *testing.T) {
	repo := employee.NewEmployeeRepository(DB)
	tx, err := repo.BeginTransaction(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	err = tx.Rollback()
	assert.NoError(t, err)
}

func TestFindByNameTx_Exists(t *testing.T) {
	repo := employee.NewEmployeeRepository(DB)

	clearTables()

	tx, err := repo.BeginTransaction(context.Background())
	if err != nil {
		t.Fatalf("Error beginning transaction: %v", err)
	}

	// Insert role inside test
	var roleID int64 = 1
	err = DB.QueryRow(`INSERT INTO role (name) VALUES ($1) RETURNING id`, "Test Role").Scan(&roleID)
	assert.NoError(t, err)

	empl := &employee.Entity{
		Name:       "John Doe",
		Email:      "john@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     roleID,
	}

	// Insert a test entity
	_, err = tx.Exec("INSERT INTO employee (name, email, position, department, role_id) VALUES ($1, $2, $3, $4, $5)",
		empl.Name, empl.Email, empl.Position, empl.Department, empl.RoleId)
	assert.NoError(t, err)

	exists, err := repo.FindByNameTx(context.Background(), tx, "John Doe")
	assert.NoError(t, err)
	assert.True(t, exists)
	err = tx.Commit()
	assert.NoError(t, err)
}

func TestFindByNameTx_NotExists(t *testing.T) {
	clearTables()
	repo := employee.NewEmployeeRepository(DB)
	tx, err := repo.BeginTransaction(context.Background())
	assert.NoError(t, err)

	exists, err := repo.FindByNameTx(context.Background(), tx, "NonExistentName")
	assert.NoError(t, err)
	assert.False(t, exists)
	err = tx.Commit()
	assert.NoError(t, err)
}

func TestSaveTx(t *testing.T) {
	repo := employee.NewEmployeeRepository(DB)

	clearTables()

	tx, err := repo.BeginTransaction(context.Background())
	if err != nil {
		t.Fatalf("Error beginning transaction: %v", err)
	}

	var roleID int64 = 1
	err = DB.QueryRow(`INSERT INTO role (name) VALUES ($1) RETURNING id`, "Test Role").Scan(&roleID)
	assert.NoError(t, err)

	employee := &employee.Entity{
		Name:       "John Doe",
		Email:      "john@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     roleID,
	}

	id, err := repo.SaveTx(context.Background(), tx, *employee)
	assert.NoError(t, err)
	assert.NotZero(t, id)
	err = tx.Commit()
	assert.NoError(t, err)

	var retrievedName string
	err = DB.QueryRow("SELECT name FROM employee WHERE id = $1", id).Scan(&retrievedName)
	assert.NoError(t, err)
	assert.Equal(t, "John Doe", retrievedName)
}

// Проверка всей логику постраничного получения данных
func TestEmployeePaginationIntegration(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	// Сначала создаем тестовые роли
	createTestRoles(t)

	// Создаем 5 тестовых записей сотрудников
	employees := createTestEmployees(t, 5)
	require.Len(t, employees, 5, "Should create exactly 5 employees")

	t.Run("First page with 3 records", func(t *testing.T) {
		// Запрашиваем первую страницу с 3 записями
		req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=1&pageSize=3", nil)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response common.Response[PageResponse]
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Проверяем результат
		data := response.Data
		assert.Equal(t, 1, data.PageNumber)
		assert.Equal(t, 3, data.PageSize)
		assert.Equal(t, int64(5), data.TotalCount)
		assert.Equal(t, 2, data.TotalPages) // (5 + 3 - 1) / 3 = 2
		assert.Len(t, data.Data, 3, "Should return exactly 3 records")
	})

	t.Run("Second page with 3 records", func(t *testing.T) {
		// Запрашиваем вторую страницу с 3 записями
		req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=2&pageSize=3", nil)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response common.Response[PageResponse]
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response.Data
		assert.Equal(t, 2, data.PageNumber)
		assert.Equal(t, 3, data.PageSize)
		assert.Equal(t, int64(5), data.TotalCount)
		assert.Equal(t, 2, data.TotalPages)
		assert.Len(t, data.Data, 2, "Should return exactly 2 records")
	})

	t.Run("Third page with 3 records", func(t *testing.T) {
		// Запрашиваем третью страницу с 3 записями
		req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=3&pageSize=3", nil)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response common.Response[PageResponse]
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response.Data
		assert.Equal(t, 3, data.PageNumber)
		assert.Equal(t, 3, data.PageSize)
		assert.Equal(t, int64(5), data.TotalCount)
		assert.Equal(t, 2, data.TotalPages)
		assert.Len(t, data.Data, 0, "Should return 0 records")
	})

	t.Run("Invalid web request", func(t *testing.T) {
		// Направляем невалидный запрос с некорректными параметрами
		req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=invalid&pageSize=abc", nil)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResponse common.Response[any]
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		require.NoError(t, err)

		assert.Contains(t, errorResponse.Message, "Invalid pageNumber parameter")
		assert.False(t, errorResponse.Success)
	})

	t.Run("Request without pageNumber parameter", func(t *testing.T) {
		// Ожидаемое поведение: использование значения по умолчанию (1)
		req := httptest.NewRequest("GET", "/api/v1/employees/page?pageSize=3", nil)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response common.Response[PageResponse]
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)

		// Проверяем, что использовано значение по умолчанию
		data := response.Data
		assert.Equal(t, 1, data.PageNumber, "Should use default pageNumber=1")
		assert.Equal(t, 3, data.PageSize)
		assert.Len(t, data.Data, 3, "Should return 3 records from first page")
	})

	t.Run("Request without pageSize parameter", func(t *testing.T) {
		// Ожидаемое поведение: использование значения по умолчанию (10)
		req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=1", nil)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response common.Response[PageResponse]
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)

		// Проверяем, что использовано значение по умолчанию
		data := response.Data
		assert.Equal(t, 1, data.PageNumber)
		assert.Equal(t, 10, data.PageSize, "Should use default pageSize=10")
		assert.Len(t, data.Data, 5, "Should return all 5 records since pageSize=10 > totalCount=5")
	})

	t.Run("Edge case: pageNumber=0", func(t *testing.T) {
		// Проверяем обработку граничного случая с pageNumber=0
		req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=0&pageSize=3", nil)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResponse common.Response[PageResponse]
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		require.NoError(t, err)

		assert.False(t, errorResponse.Success)
		assert.Contains(t, errorResponse.Message, "Error when getting paginated employees")
	})

	t.Run("Edge case: pageSize=0", func(t *testing.T) {
		// Проверяем обработку граничного случая с pageSize=0
		req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=1&pageSize=0", nil)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResponse common.Response[PageResponse]
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		require.NoError(t, err)

		assert.False(t, errorResponse.Success)
		assert.Contains(t, errorResponse.Message, "Error when getting paginated employees")
	})

	t.Run("Edge case: pageSize > 100", func(t *testing.T) {
		// Проверяем обработку граничного случая с pageSize > 100
		req := httptest.NewRequest("GET", "/api/v1/employees/page?pageNumber=1&pageSize=101", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResponse common.Response[PageResponse]
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		require.NoError(t, err)

		assert.False(t, errorResponse.Success)
		assert.Contains(t, errorResponse.Message, "Error when getting paginated employees")
	})
}

func TestFindEmployeesWithPagination_NoTextFilter(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	// Сначала создаем тестовые роли
	createTestRoles(t)

	// Создаем 7 тестовых записей сотрудников
	employees := createTestEmployees(t, 7)
	require.Len(t, employees, 7, "Should create exactly 7 employees")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees/page?pageNumber=1&pageSize=5", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response common.Response[PageResponse]
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, 1, response.Data.PageNumber)
	assert.Equal(t, 5, response.Data.PageSize)
	assert.Equal(t, int64(7), response.Data.TotalCount)
	assert.Equal(t, 2, response.Data.TotalPages)
	assert.Len(t, response.Data.Data, 5)
}

func TestFindEmployeesWithPagination_EmptyTextFilter(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	createTestRoles(t)

	employees := createTestEmployees(t, 7)
	require.Len(t, employees, 7, "Should create exactly 7 employees")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees/page?pageNumber=1&pageSize=5&textFilter=", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response common.Response[PageResponse]
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, int64(7), response.Data.TotalCount)
	assert.Len(t, response.Data.Data, 5)
}

func TestFindEmployeesWithPagination_WhitespaceOnlyTextFilter(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	createTestRoles(t)

	employees := createTestEmployees(t, 7)
	require.Len(t, employees, 7, "Should create exactly 7 employees")

	testCases := []struct {
		name       string
		textFilter string
	}{
		{"Spaces only", "   "},
		{"Tabs only", "\t\t\t"},
		{"Newlines only", "\n\n"},
		{"Mixed whitespace", " \t\n "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := url.Values{}
			params.Add("pageNumber", "1")
			params.Add("pageSize", "7")
			params.Add("textFilter", tc.textFilter)

			url := fmt.Sprintf("/api/v1/employees/page?%s", params.Encode())
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)

			defer func() {
				_ = resp.Body.Close()
			}()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response common.Response[PageResponse]
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			assert.True(t, response.Success)
			assert.Equal(t, int64(7), response.Data.TotalCount) // Все записи, фильтр не применяется
		})
	}
}

func TestFindEmployeesWithPagination_TextFilterLessThan3Chars(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	createTestRoles(t)

	employees := createTestEmployees(t, 6)
	require.Len(t, employees, 6, "Should create exactly 6 employees")

	testCases := []struct {
		name       string
		textFilter string
	}{
		{"1 character", "M"},
		{"2 characters", "Ma"},
		{"2 chars with spaces", " J "},
		{"1 char with whitespace", "\tE\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := url.Values{}
			params.Add("pageNumber", "1")
			params.Add("pageSize", "7")
			params.Add("textFilter", tc.textFilter)

			url := fmt.Sprintf("/api/v1/employees/page?%s", params.Encode())
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)

			defer func() {
				_ = resp.Body.Close()
			}()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response common.Response[PageResponse]
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			assert.True(t, response.Success)
			assert.Equal(t, int64(6), response.Data.TotalCount)
		})
	}
}

func TestFindEmployeesWithPagination_TextFilterExactMatch(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	createTestRoles(t)

	employees := createTestEmployees(t, 10)
	require.Len(t, employees, 10, "Should create exactly 10 employees")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees/page?pageNumber=1&pageSize=10&textFilter=Jane", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response common.Response[PageResponse]
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, int64(5), response.Data.TotalCount)
	assert.Equal(t, 1, response.Data.TotalPages)
	assert.Equal(t, 5, len(response.Data.Data))

	// Проверяем, что все найденные записи содержат "Jane"
	for _, emp := range response.Data.Data {
		assert.Contains(t, emp.Name, "Jane")
	}
}

func TestFindEmployeesWithPagination_TextFilterPartialMatch(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	createTestRoles(t)

	employees := createTestEmployees(t, 10)
	require.Len(t, employees, 10, "Should create exactly 10 employees")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees/page?pageNumber=1&pageSize=10&textFilter=ane", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response common.Response[PageResponse]
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.Success)

	assert.Equal(t, int64(5), response.Data.TotalCount)
	assert.Equal(t, 1, response.Data.TotalPages)
	assert.Equal(t, 5, len(response.Data.Data))
}

func TestFindEmployeesWithPagination_TextFilterNoMatch(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	createTestRoles(t)

	employees := createTestEmployees(t, 10)
	require.Len(t, employees, 10, "Should create exactly 10 employees")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees/page?pageNumber=1&pageSize=10&textFilter=Нет", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response common.Response[PageResponse]
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, int64(0), response.Data.TotalCount)
	assert.Equal(t, 0, response.Data.TotalPages)
	assert.Len(t, response.Data.Data, 0)
}

func TestFindEmployeesWithPagination_TextFilterCaseInsensitive(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	createTestRoles(t)

	employees := createTestEmployees(t, 20)
	require.Len(t, employees, 20, "Should create exactly 20 employees")
	testCases := []struct {
		name       string
		textFilter string
		expected   int64
	}{
		{"Lowercase", "jane", 10},
		{"Mixed case", "JaNe", 10},
		{"Lowercase partial", "mar", 10},
		{"Uppercase partial", "MAR", 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/employees/page?pageNumber=1&pageSize=20&textFilter=%s", tc.textFilter)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)

			defer func() {
				_ = resp.Body.Close()
			}()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response common.Response[PageResponse]
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			assert.True(t, response.Success)
			assert.Equal(t, tc.expected, response.Data.TotalCount)
		})
	}
}

func TestFindEmployeesWithPagination_TextFilterWithPagination(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	createTestRoles(t)

	createTestEmployees(t, 12)

	// Первая страница
	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees/page?pageNumber=1&pageSize=2&textFilter=Jane", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response common.Response[PageResponse]
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, 1, response.Data.PageNumber)
	assert.Equal(t, 2, response.Data.PageSize)
	assert.Equal(t, int64(6), response.Data.TotalCount)
	assert.Equal(t, 3, response.Data.TotalPages) // 6/2 = 3 страницы
	assert.Len(t, response.Data.Data, 2)         // На странице 2 записи

	// Вторая страница
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/employees/page?pageNumber=2&pageSize=2&textFilter=Jane", nil)
	req.Header.Set("Content-Type", "application/json")
	resp2, err := app.Test(req2)
	require.NoError(t, err)

	defer func() {
		_ = resp2.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var response2 common.Response[PageResponse]
	err = json.NewDecoder(resp2.Body).Decode(&response2)
	require.NoError(t, err)

	assert.True(t, response2.Success)
	assert.Equal(t, 2, response2.Data.PageNumber)
	assert.Len(t, response2.Data.Data, 2)
}

func TestFindEmployeesWithPagination_TextFilterWithSpaces(t *testing.T) {
	clearTables()

	app := appLaunchKit()

	createTestRoles(t)

	employees := createTestEmployees(t, 10)
	require.Len(t, employees, 10, "Should create exactly 10 employees")

	params := url.Values{}
	params.Add("pageNumber", "1")
	params.Add("pageSize", "10")
	params.Add("textFilter", "  Mari  ")

	url := fmt.Sprintf("/api/v1/employees/page?%s", params.Encode())
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response common.Response[PageResponse]
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, int64(0), response.Data.TotalCount)
}

// создает тестовые роли для сотрудников
func createTestRoles(t *testing.T) []int64 {
	roles := []struct {
		name        string
		description string
		status      bool
		parentId    *int64
	}{
		{"Developer", "Software Developer role with programming responsibilities", true, nil},
		{"Manager", "Team Manager role with leadership responsibilities", true, nil},
		{"Analyst", "Business Analyst role with analytical responsibilities", true, nil},
	}

	var roleIDs []int64
	for _, role := range roles {
		var roleID int64
		query := `INSERT INTO role (name, description, status, parent_id) VALUES ($1, $2, $3, $4) RETURNING id`
		err := DB.Get(&roleID, query, role.name, role.description, role.status, role.parentId)
		require.NoError(t, err)
		roleIDs = append(roleIDs, roleID)
	}

	return roleIDs
}

// создает указанное количество тестовых сотрудников
func createTestEmployees(t *testing.T, count int) []int64 {
	// Сначала получаем ID существующих ролей
	var roleIDs []int64
	err := DB.Select(&roleIDs, "SELECT id FROM role WHERE status = true LIMIT 3")
	require.NoError(t, err)
	require.NotEmpty(t, roleIDs, "Should have at least one active role")

	departments := []string{"IT", "HR", "Finance", "Marketing", "Operations"}
	positions := []string{"Developer", "Analyst", "Manager", "Specialist", "Coordinator"}

	var employeeIDs []int64

	half := count / 2

	for i := range count {
		var firstName string
		if i < half {
			firstName = "Mari"
		} else {
			firstName = "Jane"
		}
		employee := struct {
			name       string
			email      string
			position   string
			department string
			roleID     int64
		}{
			// генерация реалистичных данных для тестовых сотрудников
			name:       fmt.Sprintf("%s %s", firstName, fake.LastName()),
			email:      fake.EmailAddress(),
			position:   positions[i%len(positions)],
			department: departments[i%len(departments)],
			roleID:     roleIDs[i%len(roleIDs)],
		}

		var employeeID int64
		query := `INSERT INTO employee (name, email, position, department, role_id) 
				  VALUES ($1, $2, $3, $4, $5) RETURNING id`
		err := DB.Get(&employeeID, query, employee.name, employee.email, employee.position, employee.department, employee.roleID)
		require.NoError(t, err)
		employeeIDs = append(employeeIDs, employeeID)
	}

	return employeeIDs
}

// PageResponse структура для десериализации ответа
type PageResponse struct {
	Data       []Response `json:"data"`
	PageNumber int        `json:"pageNumber"`
	PageSize   int        `json:"pageSize"`
	TotalCount int64      `json:"totalCount"`
	TotalPages int        `json:"totalPages"`
}

// Response структура для десериализации данных сотрудника
type Response struct {
	Id         int64     `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Position   string    `json:"position"`
	Department string    `json:"department"`
	RoleId     int64     `json:"role_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
