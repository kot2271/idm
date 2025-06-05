package role

import (
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Repository struct {
	db *sqlx.DB
}

func NewRoleRepository(database *sqlx.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) FindById(id int64) (role Entity, err error) {
	err = r.db.Get(&role, "SELECT * FROM role WHERE id = $1", id)
	return role, err
}

func (r *Repository) Add(role *Entity) error {
	err := r.db.QueryRow(
		"INSERT INTO role (name, description, status, parent_id) VALUES ($1, $2, $3, $4) RETURNING id",
		role.Name, role.Desc, role.Status, role.ParentId,
	).Scan(&role.Id)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) FindAll() ([]Entity, error) {
	var roles []Entity
	err := r.db.Select(&roles, "SELECT * FROM role")
	return roles, err
}

func (r *Repository) FindByIds(ids []int64) ([]Entity, error) {
	var roles []Entity
	if len(ids) == 0 {
		return roles, nil
	}
	err := r.db.Select(&roles, "SELECT * FROM role WHERE id = ANY ($1)", pq.Array(ids))
	return roles, err
}

func (r *Repository) DeleteById(id int64) error {
	_, err := r.db.Exec("DELETE FROM role WHERE id = $1", id)
	return err
}

func (r *Repository) DeleteByIds(ids []int64) error {
	_, err := r.db.Exec("DELETE FROM role WHERE id = ANY ($1)", pq.Array(ids))
	return err
}
