package employee

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type EmployeeRepository struct {
	db *sqlx.DB
}

func NewEmployeeRepository(database *sqlx.DB) *EmployeeRepository {
	return &EmployeeRepository{db: database}
}

type EmployeeEntity struct {
	Id         int64     `db:"id"`
	Name       string    `db:"name"`
	Email      string    `db:"email"`
	Position   string    `db:"position"`
	Department string    `db:"department"`
	RoleId     int64     `db:"role_id"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (r *EmployeeRepository) FindById(id int64) (employee EmployeeEntity, err error) {
	err = r.db.Get(&employee, "SELECT * FROM employee WHERE id = $1", id)
	return employee, err
}

func (r *EmployeeRepository) Add(employee *EmployeeEntity) error {
	err := r.db.QueryRow(
		"INSERT INTO employee (name, email, position, department, role_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		employee.Name, employee.Email, employee.Position, employee.Department, employee.RoleId,
	).Scan(&employee.Id)
	if err != nil {
		return err
	}
	return nil
}

func (r *EmployeeRepository) FindAll() ([]EmployeeEntity, error) {
	var employees []EmployeeEntity
	err := r.db.Select(&employees, "SELECT * FROM employee")
	return employees, err
}

func (r *EmployeeRepository) FindByIds(ids []int64) ([]EmployeeEntity, error) {
	var employees []EmployeeEntity
	if len(ids) == 0 {
		return employees, nil
	}
	err := r.db.Select(&employees, "SELECT * FROM employee WHERE id = ANY ($1)", pq.Array(ids))
	return employees, err
}

func (r *EmployeeRepository) DeleteById(id int64) error {
	_, err := r.db.Exec("DELETE FROM employee WHERE id = $1", id)
	return err
}

func (r *EmployeeRepository) DeleteByIds(ids []int64) error {
	_, err := r.db.Exec("DELETE FROM employee WHERE id = ANY ($1)", pq.Array(ids))
	return err
}
