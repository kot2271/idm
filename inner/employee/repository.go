package employee

import (
	"context"
	"database/sql"
	"errors"

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

func (r *Repository) DeleteById(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM employee WHERE id = $1", id)
	return err
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
