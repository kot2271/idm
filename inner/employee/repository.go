package employee

import (
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
