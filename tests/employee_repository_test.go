package tests

import (
	"context"
	"testing"

	"idm/inner/employee"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

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
