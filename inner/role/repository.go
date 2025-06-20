package role

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Repository struct {
	db *sqlx.DB
}

func NewRoleRepository(database *sqlx.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) FindById(ctx context.Context, id int64) (role Entity, err error) {
	err = r.db.GetContext(ctx, &role, "SELECT * FROM role WHERE id = $1", id)
	return role, err
}

func (r *Repository) Add(ctx context.Context, role *Entity) error {
	err := r.db.QueryRowContext(
		ctx,
		"INSERT INTO role (name, description, status, parent_id) VALUES ($1, $2, $3, $4) RETURNING id",
		role.Name, role.Desc, role.Status, role.ParentId,
	).Scan(&role.Id)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) FindAll(ctx context.Context) ([]Entity, error) {
	var roles []Entity
	err := r.db.SelectContext(ctx, &roles, "SELECT * FROM role")
	return roles, err
}

func (r *Repository) FindByIds(ctx context.Context, ids []int64) ([]Entity, error) {
	var roles []Entity
	if len(ids) == 0 {
		return roles, nil
	}
	err := r.db.SelectContext(ctx, &roles, "SELECT * FROM role WHERE id = ANY ($1)", pq.Array(ids))
	return roles, err
}

func (r *Repository) DeleteById(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM role WHERE id = $1", id)
	return err
}

func (r *Repository) DeleteByIds(ctx context.Context, ids []int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM role WHERE id = ANY ($1)", pq.Array(ids))
	return err
}

// Транзакционные методы
func (r *Repository) BeginTransaction(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

// Найти роль по имени
func (r *Repository) FindByNameTx(ctx context.Context, tx *sqlx.Tx, name string) (isExists bool, err error) {
	err = tx.GetContext(
		ctx,
		&isExists,
		"select exists(select 1 from role where name = $1)",
		name,
	)
	return isExists, err
}

// Создать новую роль
func (r *Repository) SaveTx(ctx context.Context, tx *sqlx.Tx, role Entity) (roleId int64, err error) {
	err = tx.GetContext(
		ctx,
		&roleId,
		`INSERT INTO role (name, description, status, parent_id) VALUES ($1, $2, $3, $4) RETURNING id`,
		role.Name, role.Desc, role.Status, role.ParentId)
	return roleId, err
}
