package role

import (
	"context"
	"errors"
	"idm/inner/common"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
)

// Объявляем структуру мок-репозитория
type MockRepo struct {
	mock.Mock
}

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(request any) error {
	args := m.Called(request)
	return args.Error(0)
}

func (m *MockRepo) FindById(ctx context.Context, id int64) (Entity, error) {
	args := m.Called(id)
	return args.Get(0).(Entity), args.Error(1)
}

func (m *MockRepo) Add(ctx context.Context, role *Entity) error {
	args := m.Called(role)
	return args.Error(0)
}

func (m *MockRepo) FindAll(ctx context.Context) ([]Entity, error) {
	args := m.Called()
	return args.Get(0).([]Entity), args.Error(1)
}

func (m *MockRepo) FindByIds(ctx context.Context, ids []int64) ([]Entity, error) {
	args := m.Called(ids)
	return args.Get(0).([]Entity), args.Error(1)
}

func (m *MockRepo) DeleteById(ctx context.Context, id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockRepo) DeleteByIds(ctx context.Context, ids []int64) error {
	args := m.Called(ids)
	return args.Error(0)
}

func (m *MockRepo) BeginTransaction(ctx context.Context) (*sqlx.Tx, error) {
	args := m.Called()
	var tx *sqlx.Tx
	if val := args.Get(0); val != nil {
		tx = val.(*sqlx.Tx)
	}
	return tx, args.Error(1)
}

func (m *MockRepo) FindByNameTx(ctx context.Context, tx *sqlx.Tx, name string) (bool, error) {
	args := m.Called(tx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepo) SaveTx(ctx context.Context, tx *sqlx.Tx, role Entity) (int64, error) {
	args := m.Called(tx, role)
	return args.Get(0).(int64), args.Error(1)
}

// логгер для тестов
func createTestLogger() *common.Logger {
	cfg := common.Config{
		DbDriverName:   "postgres",
		Dsn:            "localhost port=5432 user=wronguser password=wrongpass dbname=postgres sslmode=disable",
		AppName:        "test_app",
		AppVersion:     "1.0.0",
		LogLevel:       "DEBUG",
		LogDevelopMode: true,
	}
	return common.NewLogger(cfg)
}

func TestService_FindById(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	parentId := int64(0)
	parentId++
	entity := Entity{
		Id:        2,
		Name:      "Admin",
		Desc:      "Administrator role",
		Status:    true,
		ParentId:  &parentId,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockRepo.On("FindById", int64(2)).Return(entity, nil)

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.FindById(context.Background(), 2)

	assert.NoError(t, err)
	assert.Equal(t, entity.toResponse(), result)
	mockRepo.AssertExpectations(t)
}

func TestService_FindById_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("FindById", int64(2)).Return(Entity{}, errors.New("db error"))

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.FindById(context.Background(), 2)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	mockRepo.AssertExpectations(t)
}

func TestService_Add(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	parentId := int64(1)
	parentId++
	entity := &Entity{
		Name:      "User",
		Desc:      "Regular user role",
		Status:    true,
		ParentId:  &parentId,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockRepo.On("Add", entity).Return(nil)

	svc := NewService(mockRepo, validator, logger)

	validator.On("Validate", entity).Return(nil)

	result, err := svc.Add(context.Background(), entity)

	assert.NoError(t, err)
	assert.Equal(t, entity.toResponse(), result)
	mockRepo.AssertExpectations(t)
}

func TestService_Add_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("Add", mock.Anything).Return(errors.New("db error"))

	svc := NewService(mockRepo, validator, logger)

	validator.On("Validate", mock.Anything).Return(nil)

	result, err := svc.Add(context.Background(), &Entity{})

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	mockRepo.AssertExpectations(t)
}

func TestService_FindAll(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	rootId := int64(0)
	adminId := rootId + 1
	userId := adminId + 1
	entities := []Entity{
		{
			Id:        2,
			Name:      "Admin",
			Desc:      "Administrator role",
			Status:    true,
			ParentId:  &adminId,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Id:        3,
			Name:      "User",
			Desc:      "Regular user role",
			Status:    true,
			ParentId:  &userId,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	mockRepo.On("FindAll", mock.Anything).Return(entities, nil)

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.FindAll(context.Background())

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, entities[0].toResponse(), result[0])
	assert.Equal(t, entities[1].toResponse(), result[1])
	mockRepo.AssertExpectations(t)
}

func TestService_FindAll_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("FindAll", mock.Anything).Return([]Entity{}, errors.New("db error"))

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.FindAll(context.Background())

	assert.Error(t, err)
	assert.Equal(t, 0, len(result))
	mockRepo.AssertExpectations(t)
}

func TestService_FindByIds(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	ids := []int64{2, 3}
	rootId := int64(0)
	adminId := rootId + 1
	userId := adminId + 1
	entities := []Entity{
		{
			Id:        2,
			Name:      "Admin",
			Desc:      "Administrator role",
			Status:    true,
			ParentId:  &adminId,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Id:        3,
			Name:      "User",
			Desc:      "Regular user role",
			Status:    true,
			ParentId:  &userId,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	mockRepo.On("FindByIds", ids).Return(entities, nil)

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.FindByIds(context.Background(), ids)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, entities[0].toResponse(), result[0])
	assert.Equal(t, entities[1].toResponse(), result[1])
	mockRepo.AssertExpectations(t)
}

func TestService_FindByIds_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("FindByIds", []int64{1, 2}).Return([]Entity{}, errors.New("db error"))

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.FindByIds(context.Background(), []int64{1, 2})

	assert.Error(t, err)
	assert.Equal(t, 0, len(result))
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteById(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("DeleteById", int64(2)).Return(nil)

	svc := NewService(mockRepo, validator, logger)

	err := svc.DeleteById(context.Background(), 2)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteById_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("DeleteById", int64(1)).Return(errors.New("db error"))

	svc := NewService(mockRepo, validator, logger)

	err := svc.DeleteById(context.Background(), 1)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteByIds(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("DeleteByIds", []int64{2, 3}).Return(nil)

	svc := NewService(mockRepo, validator, logger)

	err := svc.DeleteByIds(context.Background(), []int64{2, 3})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteByIds_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("DeleteByIds", []int64{1, 2}).Return(errors.New("db error"))

	svc := NewService(mockRepo, validator, logger)

	err := svc.DeleteByIds(context.Background(), []int64{1, 2})

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

// Тесты для метода CreateRole
func TestService_CreateRole(t *testing.T) {
	t.Run("Successful role creation", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(t, err)

		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "TestRole",
			Description: "Test Description",
			Status:      true,
			ParentId:    nil,
		}

		expectedRoleId := int64(123)

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction", mock.Anything).Return(tx, nil)
		mockRepo.On("FindByNameTx", tx, "TestRole").Return(false, nil)
		mockRepo.On("SaveTx", tx, request.ToEntity()).Return(expectedRoleId, nil)

		roleId, err := service.CreateRole(context.Background(), request)

		assert.NoError(t, err)
		assert.Equal(t, expectedRoleId, roleId)

		mockValidator.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Validation error", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()
		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "", // невалидное имя
			Description: "Test Description",
			Status:      true,
		}

		validationError := errors.New("name is required")

		mockValidator.On("Validate", request).Return(validationError)

		roleId, err := service.CreateRole(context.Background(), request)

		assert.Error(t, err)
		assert.Equal(t, int64(0), roleId)
		assert.IsType(t, common.RequestValidationError{}, err)
		assert.Contains(t, err.Error(), "name is required")

		mockValidator.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "BeginTransaction")
	})

	t.Run("Transaction creation error", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(t, err)

		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "TestRole",
			Description: "Test Description",
			Status:      true,
		}

		transactionError := errors.New("database connection error")

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction", mock.Anything).Return(tx, transactionError)

		roleId, err := service.CreateRole(context.Background(), request)

		assert.Error(t, err)
		assert.Equal(t, int64(0), roleId)
		assert.Contains(t, err.Error(), "error create role: error creating transaction")
		assert.Contains(t, err.Error(), "database connection error")

		mockValidator.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("A role with this name already exists", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(t, err)

		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "ExistingRole",
			Description: "Test Description",
			Status:      true,
		}

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction").Return(tx, nil)
		mockRepo.On("FindByNameTx", tx, "ExistingRole").Return(true, nil)

		roleId, err := service.CreateRole(context.Background(), request)

		assert.Error(t, err)
		assert.Equal(t, int64(0), roleId)

		var alreadyExistsErr common.AlreadyExistsError
		assert.True(t, errors.As(err, &alreadyExistsErr))
		assert.Contains(t, err.Error(), "role with name ExistingRole already exists")

		mockValidator.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "SaveTx")
		assert.Error(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Error when searching for a role by name", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(t, err)

		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "TestRole",
			Description: "Test Description",
			Status:      true,
		}

		findError := errors.New("database query error")

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction").Return(tx, nil)
		mockRepo.On("FindByNameTx", tx, "TestRole").Return(false, findError)

		roleId, err := service.CreateRole(context.Background(), request)

		assert.Error(t, err)
		assert.Equal(t, int64(0), roleId)
		assert.Contains(t, err.Error(), "error finding role by name: TestRole")
		assert.Contains(t, err.Error(), "database query error")

		mockValidator.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "SaveTx")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Error saving a role", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(t, err)

		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "TestRole",
			Description: "Test Description",
			Status:      true,
		}

		saveError := errors.New("insert failed")

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction").Return(tx, nil)
		mockRepo.On("FindByNameTx", tx, "TestRole").Return(false, nil)
		mockRepo.On("SaveTx", tx, request.ToEntity()).Return(int64(0), saveError)

		roleId, err := service.CreateRole(context.Background(), request)

		assert.Error(t, err)
		assert.Equal(t, int64(0), roleId)
		assert.Contains(t, err.Error(), "error creating role with name: TestRole")
		assert.Contains(t, err.Error(), "insert failed")

		mockValidator.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Creating a role with ParentId", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(t, err)

		service := NewService(mockRepo, mockValidator, logger)

		parentId := int64(456)
		request := CreateRequest{
			Name:        "ChildRole",
			Description: "Child Role Description",
			Status:      true,
			ParentId:    &parentId,
		}

		expectedRoleId := int64(789)

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction").Return(tx, nil)
		mockRepo.On("FindByNameTx", tx, "ChildRole").Return(false, nil)
		mockRepo.On("SaveTx", tx, request.ToEntity()).Return(expectedRoleId, nil)

		roleId, err := service.CreateRole(context.Background(), request)

		assert.NoError(t, err)
		assert.Equal(t, expectedRoleId, roleId)

		mockValidator.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}

// Бенчмарк тесты для метода CreateRole
func BenchmarkService_CreateRole(b *testing.B) {
	b.Run("Successful role creation", func(b *testing.B) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(b, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(b, err)
		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "BenchmarkRole",
			Description: "Benchmark Description",
			Status:      true,
			ParentId:    nil,
		}

		expectedRoleId := int64(123)

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction").Return(tx, nil)
		mockRepo.On("FindByNameTx", tx, "BenchmarkRole").Return(false, nil)
		mockRepo.On("SaveTx", tx, request.ToEntity()).Return(expectedRoleId, nil)

		// Сброс таймера, чтобы исключить время подготовки
		b.ResetTimer()

		for b.Loop() {
			_, err := service.CreateRole(context.Background(), request)
			if err != nil {
				b.Fatalf("Unexpected error during benchmark: %v", err)
			}
		}
	})

	b.Run("Validation error", func(b *testing.B) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "", // невалидное имя
			Description: "Benchmark Description",
			Status:      true,
		}

		validationError := errors.New("name is required")
		mockValidator.On("Validate", request).Return(validationError)

		b.ResetTimer()

		for b.Loop() {
			_, err := service.CreateRole(context.Background(), request)
			if err == nil {
				b.Fatal("Expected validation error, got nil")
			}
		}
	})

	b.Run("The role already exists", func(b *testing.B) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(b, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(b, err)
		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "ExistingBenchmarkRole",
			Description: "Benchmark Description",
			Status:      true,
		}

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction").Return(tx, nil)
		mockRepo.On("FindByNameTx", tx, "ExistingBenchmarkRole").Return(true, nil)

		b.ResetTimer()

		for b.Loop() {
			_, err := service.CreateRole(context.Background(), request)
			if err == nil {
				b.Fatal("Expected AlreadyExistsError, got nil")
			}
		}
	})

	b.Run("Creating a role with ParentId", func(b *testing.B) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(b, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(b, err)
		service := NewService(mockRepo, mockValidator, logger)

		parentId := int64(456)
		request := CreateRequest{
			Name:        "ChildBenchmarkRole",
			Description: "Child Benchmark Description",
			Status:      true,
			ParentId:    &parentId,
		}

		expectedRoleId := int64(789)

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction").Return(tx, nil)
		mockRepo.On("FindByNameTx", tx, "ChildBenchmarkRole").Return(false, nil)
		mockRepo.On("SaveTx", tx, request.ToEntity()).Return(expectedRoleId, nil)

		b.ResetTimer()

		for b.Loop() {
			_, err := service.CreateRole(context.Background(), request)
			if err != nil {
				b.Fatalf("Unexpected error during benchmark: %v", err)
			}
		}
	})
}

// Бенчмарк для измерения аллокаций памяти
func BenchmarkService_CreateRole_Memory(b *testing.B) {
	mockRepo := new(MockRepo)
	mockValidator := new(MockValidator)
	logger := createTestLogger()

	db, sqlMock, err := sqlmock.New()
	assert.NoError(b, err)
	defer func() {
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "postgres")
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	tx, err := sqlxDB.Beginx()
	assert.NoError(b, err)
	service := NewService(mockRepo, mockValidator, logger)

	request := CreateRequest{
		Name:        "MemoryBenchmarkRole",
		Description: "Memory Benchmark Description",
		Status:      true,
		ParentId:    nil,
	}

	expectedRoleId := int64(123)

	mockValidator.On("Validate", request).Return(nil)
	mockRepo.On("BeginTransaction").Return(tx, nil)
	mockRepo.On("FindByNameTx", tx, "MemoryBenchmarkRole").Return(false, nil)
	mockRepo.On("SaveTx", tx, request.ToEntity()).Return(expectedRoleId, nil)

	// Включение отчета об аллокациях памяти
	b.ReportAllocs()

	for b.Loop() {
		_, err := service.CreateRole(context.Background(), request)
		if err != nil {
			b.Fatalf("Unexpected error during benchmark: %v", err)
		}
	}
}

// Параллельный бенчмарк для проверки производительности при конкурентном доступе
func BenchmarkService_CreateRole_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		mockRepo := new(MockRepo)
		mockValidator := new(MockValidator)
		logger := createTestLogger()

		db, sqlMock, err := sqlmock.New()
		assert.NoError(b, err)
		defer func() {
			_ = db.Close()
		}()

		sqlxDB := sqlx.NewDb(db, "postgres")
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		tx, err := sqlxDB.Beginx()
		assert.NoError(b, err)
		service := NewService(mockRepo, mockValidator, logger)

		request := CreateRequest{
			Name:        "ParallelBenchmarkRole",
			Description: "Parallel Benchmark Description",
			Status:      true,
			ParentId:    nil,
		}

		expectedRoleId := int64(123)

		mockValidator.On("Validate", request).Return(nil)
		mockRepo.On("BeginTransaction").Return(tx, nil)
		mockRepo.On("FindByNameTx", tx, "ParallelBenchmarkRole").Return(false, nil)
		mockRepo.On("SaveTx", tx, request.ToEntity()).Return(expectedRoleId, nil)

		for pb.Next() {
			_, err := service.CreateRole(context.Background(), request)
			if err != nil {
				b.Fatalf("Unexpected error during parallel benchmark: %v", err)
			}
		}
	})
}

// Бенчмарк с различными размерами данных
func BenchmarkService_CreateRole_DataSizes(b *testing.B) {
	dataSizes := []struct {
		name    string
		nameLen int
		descLen int
	}{
		{"Small", 10, 50},
		{"Medium", 50, 200},
		{"Large", 100, 500},
		{"ExtraLarge", 155, 1000}, // максимальная длина имени согласно валидации
	}

	for _, size := range dataSizes {
		b.Run(size.name, func(b *testing.B) {
			mockRepo := new(MockRepo)
			mockValidator := new(MockValidator)
			logger := createTestLogger()

			db, sqlMock, err := sqlmock.New()
			assert.NoError(b, err)
			defer func() {
				_ = db.Close()
			}()

			sqlxDB := sqlx.NewDb(db, "postgres")
			sqlMock.ExpectBegin()
			sqlMock.ExpectRollback()

			tx, err := sqlxDB.Beginx()
			assert.NoError(b, err)
			service := NewService(mockRepo, mockValidator, logger)

			// Генерация строки нужной длины
			name := generateString(size.nameLen)
			description := generateString(size.descLen)

			request := CreateRequest{
				Name:        name,
				Description: description,
				Status:      true,
				ParentId:    nil,
			}

			expectedRoleId := int64(123)

			mockValidator.On("Validate", request).Return(nil)
			mockRepo.On("BeginTransaction").Return(tx, nil)
			mockRepo.On("FindByNameTx", tx, name).Return(false, nil)
			mockRepo.On("SaveTx", tx, request.ToEntity()).Return(expectedRoleId, nil)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, err := service.CreateRole(context.Background(), request)
				if err != nil {
					b.Fatalf("Unexpected error during benchmark: %v", err)
				}
			}
		})
	}
}

// Вспомогательный метод для генерации строк заданной длины
func generateString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

func TestService_validateCreateRequest_Success(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	svc := NewService(mockRepo, validator, logger)

	parentId := int64(1)
	request := CreateRequest{
		Name:        "Admin",
		Description: "Full access",
		Status:      true,
		ParentId:    &parentId,
	}
	validator.On("Validate", request).Return(nil)

	err := svc.validateCreateRequest(request)
	assert.NoError(t, err)
	validator.AssertExpectations(t)
}

func TestService_validateCreateRequest_CustomError(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	svc := NewService(mockRepo, validator, logger)

	request := CreateRequest{Name: ""} // невалидные данные
	customErr := errors.New("custom validation error")
	validator.On("Validate", request).Return(customErr)

	err := svc.validateCreateRequest(request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom validation error")
	validator.AssertExpectations(t)
}

// Мокаем validator.ValidationErrors для проверки обработки именно этого типа
// и сообщения "Data validation error"
type mockValidationErrors struct{}

func (mockValidationErrors) Error() string { return "Data validation error" }

func TestService_validateCreateRequest_ValidationErrorsType(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	svc := NewService(mockRepo, validator, logger)

	var validationErrs = mockValidationErrors{}

	request := CreateRequest{Name: ""}
	validator.On("Validate", request).Return(validationErrs)

	err := svc.validateCreateRequest(request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Data validation error")
	validator.AssertExpectations(t)
}
