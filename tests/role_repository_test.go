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
	rootRole := &role.Entity{
		Name:   "Root",
		Desc:   "Root of all roles",
		Status: true,
		// ParentId: nil (по умолчанию nil, если поле *int64)
	}

	// Дочерняя роль Admin, ссылается на Root
	adminRole := &role.Entity{
		Name:     "Admin",
		Desc:     "Administrator role",
		Status:   true,
		ParentId: &rootRole.Id,
	}

	// Guest, ссылается на Admin
	guestRole := &role.Entity{
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
		aRole := &role.Entity{Name: "Temporary", Desc: "Test role", Status: true}
		bRole := &role.Entity{Name: "Contractor", Desc: "External role", Status: true}
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

func TestBeginTransactionRole(t *testing.T) {
	repo := role.NewRoleRepository(DB)
	tx, err := repo.BeginTransaction()
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	err = tx.Rollback()
	assert.NoError(t, err)
}

func TestFindByRoleNameTx_Exists(t *testing.T) {
	repo := role.NewRoleRepository(DB)

	clearTables()

	tx, err := repo.BeginTransaction()
	if err != nil {
		t.Fatalf("Error beginning transaction: %v", err)
	}

	rootRole := &role.Entity{
		Name:   "Root",
		Desc:   "Root of all roles",
		Status: true,
		// ParentId: nil (по умолчанию nil, если поле *int64)
	}

	err = repo.Add(rootRole)
	assert.NoError(t, err)
	assert.NotZero(t, rootRole.Id)

	role := &role.Entity{
		Name:     "Admin",
		Desc:     "Administrator role",
		Status:   true,
		ParentId: &rootRole.Id,
	}

	// Insert a test entity
	_, err = tx.Exec("INSERT INTO role (name, description, status, parent_id) VALUES ($1, $2, $3, $4)",
		role.Name, role.Desc, role.Status, role.ParentId)
	assert.NoError(t, err)

	exists, err := repo.FindByNameTx(tx, "Admin")
	assert.NoError(t, err)
	assert.True(t, exists)
	err = tx.Commit()
	assert.NoError(t, err)
}

func TestFindByRoleNameTx_NotExists(t *testing.T) {
	clearTables()
	repo := role.NewRoleRepository(DB)
	tx, err := repo.BeginTransaction()
	assert.NoError(t, err)

	exists, err := repo.FindByNameTx(tx, "NonExistentName")
	assert.NoError(t, err)
	assert.False(t, exists)
	err = tx.Commit()
	assert.NoError(t, err)
}

func TestSaveRoleTx(t *testing.T) {
	repo := role.NewRoleRepository(DB)

	clearTables()

	tx, err := repo.BeginTransaction()
	if err != nil {
		t.Fatalf("Error beginning transaction: %v", err)
	}

	rootRole := &role.Entity{
		Name:   "Root",
		Desc:   "Root of all roles",
		Status: true,
		// ParentId: nil (по умолчанию nil, если поле *int64)
	}

	err = repo.Add(rootRole)
	assert.NoError(t, err)
	assert.NotZero(t, rootRole.Id)

	role := &role.Entity{
		Name:     "Admin",
		Desc:     "Administrator role",
		Status:   true,
		ParentId: &rootRole.Id,
	}

	id, err := repo.SaveTx(tx, *role)
	assert.NoError(t, err)
	assert.NotZero(t, id)
	err = tx.Commit()
	assert.NoError(t, err)

	var retrievedName string
	err = DB.QueryRow("SELECT description FROM role WHERE id = $1", id).Scan(&retrievedName)
	assert.NoError(t, err)
	assert.Equal(t, "Administrator role", retrievedName)
}
