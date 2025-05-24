package role

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type RoleRepository struct {
	db *sqlx.DB
}

func NewRoleRepository(database *sqlx.DB) *RoleRepository {
	return &RoleRepository{db: database}
}

type RoleEntity struct {
	Id        int64     `db:"id"`
	Name      string    `db:"name"`
	Desc      string    `db:"description"`
	Status    bool      `db:"status"`
	ParentId  *int64    `db:"parent_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (r *RoleRepository) FindById(id int64) (role RoleEntity, err error) {
	err = r.db.Get(&role, "SELECT * FROM role WHERE id = $1", id)
	return role, err
}

func (r *RoleRepository) Add(role *RoleEntity) error {
	err := r.db.QueryRow(
		"INSERT INTO role (name, description, status, parent_id) VALUES ($1, $2, $3, $4) RETURNING id",
		role.Name, role.Desc, role.Status, role.ParentId,
	).Scan(&role.Id)
	if err != nil {
		return err
	}
	return nil
}

func (r *RoleRepository) FindAll() ([]RoleEntity, error) {
	var roles []RoleEntity
	err := r.db.Select(&roles, "SELECT * FROM role")
	return roles, err
}

func (r *RoleRepository) FindByIds(ids []int64) ([]RoleEntity, error) {
	var roles []RoleEntity
	if len(ids) == 0 {
		return roles, nil
	}
	err := r.db.Select(&roles, "SELECT * FROM role WHERE id = ANY ($1)", pq.Array(ids))
	return roles, err
}

func (r *RoleRepository) DeleteById(id int64) error {
	_, err := r.db.Exec("DELETE FROM role WHERE id = $1", id)
	return err
}

func (r *RoleRepository) DeleteByIds(ids []int64) error {
	_, err := r.db.Exec("DELETE FROM role WHERE id = ANY ($1)", pq.Array(ids))
	return err
}
