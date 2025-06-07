package employee

import (
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

func (r *Repository) FindById(id int64) (employee Entity, err error) {
	err = r.db.Get(&employee, "SELECT * FROM employee WHERE id = $1", id)
	return employee, err
}

func (r *Repository) Add(employee *Entity) error {
	err := r.db.QueryRow(
		"INSERT INTO employee (name, email, position, department, role_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		employee.Name, employee.Email, employee.Position, employee.Department, employee.RoleId,
	).Scan(&employee.Id)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) FindAll() ([]Entity, error) {
	var employees []Entity
	err := r.db.Select(&employees, "SELECT * FROM employee")
	return employees, err
}

func (r *Repository) FindByIds(ids []int64) ([]Entity, error) {
	var employees []Entity
	if len(ids) == 0 {
		return employees, nil
	}
	err := r.db.Select(&employees, "SELECT * FROM employee WHERE id = ANY ($1)", pq.Array(ids))
	return employees, err
}

func (r *Repository) DeleteById(id int64) error {
	_, err := r.db.Exec("DELETE FROM employee WHERE id = $1", id)
	return err
}

func (r *Repository) DeleteByIds(ids []int64) error {
	_, err := r.db.Exec("DELETE FROM employee WHERE id = ANY ($1)", pq.Array(ids))
	return err
}

// Транзакционные методы
// Создать новую транзакцию
func (r *Repository) BeginTransaction() (*sqlx.Tx, error) {
	return r.db.Beginx()
}

// Найти сотрудника по имени
func (r *Repository) FindByNameTx(tx *sqlx.Tx, name string) (isExists bool, err error) {
	err = tx.Get(
		&isExists,
		"select exists(select 1 from employee where name = $1)",
		name,
	)
	return isExists, err
}

// Создать нового сотрудника
func (r *Repository) SaveTx(tx *sqlx.Tx, employee Entity) (employeeId int64, err error) {
	err = tx.Get(
		&employeeId,
		`insert into employee (name, email, position, department, role_id) values ($1, $2, $3, $4, $5) returning id`,
		employee.Name, employee.Email, employee.Position, employee.Department, employee.RoleId)
	return employeeId, err
}

// Добавить нового сотрудника
func (r *Repository) AddWithTransaction(tx *sqlx.Tx, employee *Entity) error {
	var existing *Entity
	err := tx.Get(&existing, "SELECT id FROM employee WHERE name = $1", employee.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
		} else {
			return err
		}
	} else {
		return errors.New("employee with same name already exists")
	}

	err = tx.QueryRow(
		"INSERT INTO employee (name, email, position, department, role_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		employee.Name, employee.Email, employee.Position, employee.Department, employee.RoleId,
	).Scan(&employee.Id)

	return err
}
