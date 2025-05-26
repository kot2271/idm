package tests

import (
	"testing"

	"idm/inner/role"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestRoleRepository_CRUD(t *testing.T) {
	repo := role.NewRoleRepository(DB)

	clearTables()

	// Root role без родителя
	rootRole := &role.RoleEntity{
		Name:   "Root",
		Desc:   "Root of all roles",
		Status: true,
		// ParentId: nil (по умолчанию nil, если поле *int64)
	}

	// Дочерняя роль Admin, ссылается на Root
	adminRole := &role.RoleEntity{
		Name:     "Admin",
		Desc:     "Administrator role",
		Status:   true,
		ParentId: &rootRole.Id,
	}

	// Guest, ссылается на Admin
	guestRole := &role.RoleEntity{
		Name:     "Guest",
		Desc:     "Read-only role",
		Status:   false,
		ParentId: &adminRole.Id,
	}

	t.Run("Add", func(t *testing.T) {
		err := repo.Add(rootRole)
		assert.NoError(t, err)
		assert.NotZero(t, rootRole.Id)

		err = repo.Add(adminRole)
		assert.NoError(t, err)
		assert.NotZero(t, adminRole.Id)

		err = repo.Add(guestRole)
		assert.NoError(t, err)
		assert.NotZero(t, guestRole.Id)
	})

	t.Run("FindById", func(t *testing.T) {
		found, err := repo.FindById(adminRole.Id)
		assert.NoError(t, err)
		assert.Equal(t, "Admin", found.Name)
	})

	t.Run("FindAll", func(t *testing.T) {
		roles, err := repo.FindAll()
		assert.NoError(t, err)
		assert.Len(t, roles, 3)
		assert.Contains(t, []string{"Root", "Admin", "Guest"}, roles[0].Name)
	})

	t.Run("FindByIds", func(t *testing.T) {
		roles, err := repo.FindByIds([]int64{adminRole.Id, guestRole.Id})
		assert.NoError(t, err)
		assert.Len(t, roles, 2)
		assert.Equal(t, adminRole.Id, roles[0].Id)
	})

	t.Run("DeleteById", func(t *testing.T) {
		err := repo.DeleteById(adminRole.Id)
		assert.NoError(t, err)

		_, err = repo.FindById(adminRole.Id)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rows in result set")
	})

	t.Run("DeleteByIds", func(t *testing.T) {
		// Добавляем две роли для массового удаления
		aRole := &role.RoleEntity{Name: "Temporary", Desc: "Test role", Status: true}
		bRole := &role.RoleEntity{Name: "Contractor", Desc: "External role", Status: true}
		_ = repo.Add(aRole)
		_ = repo.Add(bRole)

		err := repo.DeleteByIds([]int64{aRole.Id, bRole.Id})
		assert.NoError(t, err)

		_, err = repo.FindById(aRole.Id)
		assert.Error(t, err)
		_, err = repo.FindById(bRole.Id)
		assert.Error(t, err)
	})
}
