package employee

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Repository struct {
	db *sqlx.DB
}

func NewEmployeeRepository(database *sqlx.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) FindById(ctx context.Context, id int64) (employee Entity, err error) {
	err = r.db.GetContext(ctx, &employee, "SELECT * FROM employee WHERE id = $1", id)
	return employee, err
}

func (r *Repository) Add(ctx context.Context, employee *Entity) error {
	err := r.db.QueryRowContext(
		ctx,
		"INSERT INTO employee (name, email, position, department, role_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		employee.Name, employee.Email, employee.Position, employee.Department, employee.RoleId,
	).Scan(&employee.Id)
	return err
}

func (r *Repository) FindAll(ctx context.Context) ([]Entity, error) {
	var employees []Entity
	err := r.db.SelectContext(ctx, &employees, "SELECT * FROM employee")
	return employees, err
}

func (r *Repository) FindByIds(ctx context.Context, ids []int64) ([]Entity, error) {
	var employees []Entity
	if len(ids) == 0 {
		return employees, nil
	}
	err := r.db.SelectContext(ctx, &employees, "SELECT * FROM employee WHERE id = ANY ($1)", pq.Array(ids))
	return employees, err
}

func (r *Repository) FindWithPagination(ctx context.Context, limit, offset int, textFilter string) ([]Entity, error) {
	var employees []Entity
	query := `SELECT * FROM employee WHERE 1 = 1`
	args := []any{limit, offset}

	// Добавляем фильтр по имени только если textFilter содержит не менее 3 не пробельных символов
	if isValidTextFilter(textFilter) {
		query += ` AND name ILIKE $3`
		args = append(args, "%"+textFilter+"%")
	}

	query += ` ORDER BY id LIMIT $1 OFFSET $2`

	err := r.db.SelectContext(ctx, &employees, query, args...)
	return employees, err
}

func (r *Repository) CountWithFilter(ctx context.Context, textFilter string) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM employee WHERE 1 = 1`
	args := []any{}

	// Фильтр по имени только если textFilter содержит не менее 3 не пробельных символов
	if isValidTextFilter(textFilter) {
		query += ` AND name ILIKE $1`
		args = append(args, "%"+textFilter+"%")
	}

	err := r.db.GetContext(ctx, &count, query, args...)
	return count, err
}

// Вспомогательная функция для проверки валидности текстового фильтра
func isValidTextFilter(textFilter string) bool {
	if textFilter == "" {
		return false
	}

	// Удаление всех пробельных символов и проверка длины
	trimmed := strings.TrimSpace(textFilter)
	nonWhitespaceCount := 0

	for _, char := range trimmed {
		if !unicode.IsSpace(char) {
			nonWhitespaceCount++
		}
	}

	return nonWhitespaceCount >= 3
}

func (r *Repository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM employee`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

func (r *Repository) DeleteById(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM employee WHERE id = $1", id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("employee with id %d not found", id)
	}

	return nil
}

func (r *Repository) DeleteByIds(ctx context.Context, ids []int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM employee WHERE id = ANY ($1)", pq.Array(ids))
	return err
}

// Транзакционные методы
// Создать новую транзакцию
func (r *Repository) BeginTransaction(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

// Найти сотрудника по имени
func (r *Repository) FindByNameTx(ctx context.Context, tx *sqlx.Tx, name string) (isExists bool, err error) {
	err = tx.GetContext(
		ctx,
		&isExists,
		"select exists(select 1 from employee where name = $1)",
		name,
	)
	return isExists, err
}

// Создать нового сотрудника
func (r *Repository) SaveTx(ctx context.Context, tx *sqlx.Tx, employee Entity) (employeeId int64, err error) {
	err = tx.GetContext(
		ctx,
		&employeeId,
		`insert into employee (name, email, position, department, role_id) values ($1, $2, $3, $4, $5) returning id`,
		employee.Name, employee.Email, employee.Position, employee.Department, employee.RoleId)
	return employeeId, err
}

// Добавить нового сотрудника
func (r *Repository) AddWithTransaction(ctx context.Context, tx *sqlx.Tx, employee *Entity) error {
	var existing *Entity
	err := tx.GetContext(ctx, &existing, "SELECT id FROM employee WHERE name = $1", employee.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
		} else {
			return err
		}
	} else {
		return errors.New("employee with same name already exists")
	}

	err = tx.QueryRowContext(
		ctx,
		"INSERT INTO employee (name, email, position, department, role_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		employee.Name, employee.Email, employee.Position, employee.Department, employee.RoleId,
	).Scan(&employee.Id)

	return err
}
