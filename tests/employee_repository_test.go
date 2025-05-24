package tests

import (
	"fmt"
	"log"
	"os"
	"testing"

	"idm/inner/employee"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "testuser"
	password = "testpass"
	dbname   = "testdb"
)

var DB *sqlx.DB

func TestMain(m *testing.M) {
	// Подключение к базе данных
	var err error
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	DB, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	// Применение миграций
	applyMigrations()

	// Запуск тестов
	code := m.Run()

	// Очистка
	os.Exit(code)
}

func applyMigrations() {
	_, err := DB.Exec(`
        CREATE TABLE IF NOT EXISTS role (
            id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
            name TEXT NOT NULL,
            description TEXT,
            status BOOLEAN DEFAULT TRUE,
            parent_id BIGINT REFERENCES role(id) ON DELETE SET NULL,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS employee (
            id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
            name TEXT NOT NULL,
            email TEXT UNIQUE NOT NULL,
            position TEXT,
            department TEXT,
            role_id BIGINT REFERENCES role(id),
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );
    `)
	if err != nil {
		log.Fatalf("Migration failed: %v\n", err)
	}
}

func clearTables() {
	_, _ = DB.Exec("DELETE FROM employee")
	_, _ = DB.Exec("DELETE FROM role")
}

func TestEmployeeRepository_CRUD(t *testing.T) {
	repo := employee.NewEmployeeRepository(DB)

	clearTables()

	// Добавляем роль
	var roleID int64 = 1
	err := DB.QueryRow(`INSERT INTO role (name) VALUES ($1) RETURNING id`, "Test Role").Scan(&roleID)
	assert.NoError(t, err)

	// Создаем сотрудника
	emp := &employee.EmployeeEntity{
		Name:       "John Doe",
		Email:      "john@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     roleID,
	}

	emp2 := &employee.EmployeeEntity{
		Name:       "Rick Sanchez",
		Email:      "rick@example.com",
		Position:   "Manager",
		Department: "IT",
		RoleId:     roleID,
	}

	t.Run("Add", func(t *testing.T) {
		err := repo.Add(emp)
		assert.NoError(t, err)
		assert.NotZero(t, emp.Id)

		err = repo.Add(emp2)
		assert.NoError(t, err)
		assert.NotZero(t, emp2.Id)
	})

	t.Run("FindById", func(t *testing.T) {
		found, err := repo.FindById(emp.Id)
		assert.NoError(t, err)
		assert.Equal(t, emp.Name, found.Name)
		assert.Equal(t, emp.Email, found.Email)
	})

	t.Run("FindAll", func(t *testing.T) {
		employees, err := repo.FindAll()
		assert.NoError(t, err)
		assert.Len(t, employees, 2)
		assert.Equal(t, emp.Email, employees[0].Email)
		assert.Equal(t, emp2.Email, employees[1].Email)
	})

	t.Run("FindByIds", func(t *testing.T) {
		employees, err := repo.FindByIds([]int64{emp.Id, emp2.Id})
		assert.NoError(t, err)
		assert.Len(t, employees, 2)
		assert.Equal(t, emp.Id, employees[0].Id)
		assert.Equal(t, emp2.Id, employees[1].Id)
	})

	t.Run("DeleteById", func(t *testing.T) {
		err := repo.DeleteById(emp.Id)
		assert.NoError(t, err)

		_, err = repo.FindById(emp.Id)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rows in result set")
	})

	t.Run("DeleteByIds", func(t *testing.T) {
		// Добавляем двух сотрудников
		e1 := &employee.EmployeeEntity{Name: "Alice", Email: "alice@example.com", RoleId: roleID}
		e2 := &employee.EmployeeEntity{Name: "Bob", Email: "bob@example.com", RoleId: roleID}
		_ = repo.Add(e1)
		_ = repo.Add(e2)

		err := repo.DeleteByIds([]int64{e1.Id, e2.Id})
		assert.NoError(t, err)

		_, err = repo.FindById(e1.Id)
		assert.Error(t, err)
		_, err = repo.FindById(e2.Id)
		assert.Error(t, err)
	})
}
