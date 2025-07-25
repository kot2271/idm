package employee

import (
	"context"
	"database/sql"
	"errors"
	"idm/inner/common"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
)

// объявляем структуру мок-репозитория
type MockRepo struct {
	mock.Mock
}

type MockTx struct {
	mock.Mock
}

// объявляем структуру Stub-репозитория
type StubRepo struct {
	entity Entity
}

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(request any) error {
	args := m.Called(request)
	return args.Error(0)
}

func (m *MockRepo) FindById(ctx context.Context, id int64) (Entity, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Entity), args.Error(1)
}

func (s *StubRepo) FindById(ctx context.Context, id int64) (Entity, error) {
	return s.entity, nil
}

func (m *MockRepo) Add(ctx context.Context, employee *Entity) error {
	args := m.Called(ctx, employee)
	return args.Error(0)
}

func (s *StubRepo) Add(ctx context.Context, entity *Entity) error {
	return nil
}

func (m *MockRepo) AddWithTransaction(ctx context.Context, tx *sqlx.Tx, employee *Entity) error {
	return m.Called(ctx, tx, employee).Error(0)
}

func (m *MockRepo) BeginTransaction(ctx context.Context) (*sqlx.Tx, error) {
	args := m.Called(ctx)
	var tx *sqlx.Tx
	if val := args.Get(0); val != nil {
		tx = val.(*sqlx.Tx)
	}
	return tx, args.Error(1)
}

func (s *StubRepo) AddWithTransaction(ctx context.Context, tx *sqlx.Tx, employee *Entity) error {
	return nil
}

func (s *StubRepo) BeginTransaction(ctx context.Context) (*sqlx.Tx, error) {
	return &sqlx.Tx{}, nil
}

func (m *MockRepo) FindAll(ctx context.Context) ([]Entity, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Entity), args.Error(1)
}

func (s *StubRepo) FindAll(ctx context.Context) ([]Entity, error) {
	return []Entity{s.entity}, nil
}

func (m *MockRepo) FindByIds(ctx context.Context, ids []int64) ([]Entity, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]Entity), args.Error(1)
}

func (s *StubRepo) FindByIds(ctx context.Context, id []int64) ([]Entity, error) {
	return []Entity{s.entity}, nil
}

func (m *MockRepo) DeleteById(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (s *StubRepo) DeleteById(ctx context.Context, id int64) error {
	return nil
}

func (m *MockRepo) DeleteByIds(ctx context.Context, ids []int64) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (s *StubRepo) DeleteByIds(ctx context.Context, id []int64) error {
	return nil
}

func (m *MockRepo) FindByNameTx(ctx context.Context, tx *sqlx.Tx, name string) (bool, error) {
	args := m.Called(ctx, tx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepo) SaveTx(ctx context.Context, tx *sqlx.Tx, employee Entity) (int64, error) {
	args := m.Called(ctx, tx, employee)
	return args.Get(0).(int64), args.Error(1)
}

func (s *StubRepo) FindByNameTx(ctx context.Context, tx *sqlx.Tx, name string) (bool, error) {
	panic("unimplemented")
}

func (s *StubRepo) SaveTx(ctx context.Context, tx *sqlx.Tx, employee Entity) (int64, error) {
	panic("unimplemented")
}

func (m *MockRepo) CountAll(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepo) FindWithPagination(ctx context.Context, limit, offset int, textFilter string) ([]Entity, error) {
	args := m.Called(ctx, limit, offset, textFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Entity), args.Error(1)
}

func (m *MockRepo) CountWithFilter(ctx context.Context, textFilter string) (int64, error) {
	args := m.Called(ctx, textFilter)
	return args.Get(0).(int64), args.Error(1)
}

func (s *StubRepo) CountAll(ctx context.Context) (int64, error) {
	panic("unimplemented")
}

func (s *StubRepo) FindWithPagination(ctx context.Context, limit int, offset int, textFilter string) ([]Entity, error) {
	panic("unimplemented")
}

func (s *StubRepo) CountWithFilter(ctx context.Context, textFilter string) (int64, error) {
	panic("unimplemented")
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

func TestService_FindById_Mock(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	entity := Entity{
		Id:         1,
		Name:       "John",
		Email:      "john@example.com",
		Position:   "Developer",
		Department: "Engineering",
		RoleId:     2,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mockRepo.On("FindById", mock.Anything, int64(1)).Return(entity, nil)

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.FindById(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, entity.toResponse(), result)
	mockRepo.AssertExpectations(t)
}

func TestService_FindById_Stub(t *testing.T) {
	logger := createTestLogger()
	stubRepo := &StubRepo{
		entity: Entity{
			Id:         1,
			Name:       "John",
			Email:      "john@example.com",
			Position:   "Developer",
			Department: "Engineering",
			RoleId:     2,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}
	validator := new(MockValidator)

	svc := NewService(stubRepo, validator, logger)

	result, err := svc.FindById(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, stubRepo.entity.toResponse(), result)
}

func TestService_FindById_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("FindById", mock.Anything, int64(1)).Return(Entity{}, errors.New("db error"))

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.FindById(context.Background(), 1)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	mockRepo.AssertExpectations(t)
}

func TestService_Add(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	entity := &Entity{
		Name:       "Jane",
		Email:      "jane@example.com",
		Position:   "Designer",
		Department: "Design",
		RoleId:     3,
	}
	mockRepo.On("Add", mock.Anything, entity).Return(nil)

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
	mockRepo.On("Add", mock.Anything, mock.Anything).Return(errors.New("db error"))

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
	entities := []Entity{
		{
			Id:         1,
			Name:       "John",
			Email:      "john@example.com",
			Position:   "Developer",
			Department: "Engineering",
			RoleId:     2,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			Id:         2,
			Name:       "Jane",
			Email:      "jane@example.com",
			Position:   "Designer",
			Department: "Design",
			RoleId:     3,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
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
	ids := []int64{1, 2}
	entities := []Entity{
		{
			Id:         1,
			Name:       "John",
			Email:      "john@example.com",
			Position:   "Developer",
			Department: "Engineering",
			RoleId:     2,
		},
		{
			Id:         2,
			Name:       "Jane",
			Email:      "jane@example.com",
			Position:   "Designer",
			Department: "Design",
			RoleId:     3,
		},
	}
	mockRepo.On("FindByIds", mock.Anything, ids).Return(entities, nil)

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
	mockRepo.On("FindByIds", mock.Anything, []int64{1, 2}).Return([]Entity{}, errors.New("db error"))

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
	mockRepo.On("DeleteById", mock.Anything, int64(1)).Return(nil)

	svc := NewService(mockRepo, validator, logger)

	err := svc.DeleteById(context.Background(), 1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteById_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("DeleteById", mock.Anything, int64(1)).Return(errors.New("db error"))

	svc := NewService(mockRepo, validator, logger)

	err := svc.DeleteById(context.Background(), 1)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteByIds(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("DeleteByIds", mock.Anything, []int64{1, 2}).Return(nil)

	svc := NewService(mockRepo, validator, logger)

	err := svc.DeleteByIds(context.Background(), []int64{1, 2})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteByIds_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("DeleteByIds", mock.Anything, []int64{1, 2}).Return(errors.New("db error"))

	svc := NewService(mockRepo, validator, logger)

	err := svc.DeleteByIds(context.Background(), []int64{1, 2})

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_BeginError(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	mockRepo.On("BeginTransaction", mock.Anything).Return(nil, errors.New("failed to begin transaction"))

	svc := NewService(mockRepo, validator, logger)

	entity := &Entity{
		Name:       "Test User",
		Email:      "test@example.com",
		Position:   "Engineer",
		Department: "Engineering",
		RoleId:     1,
	}

	result, err := svc.AddWithTransaction(context.Background(), entity)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	assert.Contains(t, err.Error(), "failed to begin")

	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_ExistingEmployeeCheckError(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	entity := &Entity{
		Name:       "Test User",
		Email:      "test@example.com",
		Position:   "Engineer",
		Department: "Engineering",
		RoleId:     1,
	}

	mockRepo.On("BeginTransaction", mock.Anything).Return((*sqlx.Tx)(nil), errors.New("transaction failed"))

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.AddWithTransaction(context.Background(), entity)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	assert.Contains(t, err.Error(), "transaction failed")
	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_EmployeeAlreadyExists(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	entity := &Entity{
		Name:       "Test User",
		Email:      "test@example.com",
		Position:   "Engineer",
		Department: "Engineering",
		RoleId:     1,
	}

	mockRepo.On("BeginTransaction", mock.Anything).Return((*sqlx.Tx)(nil), errors.New("failed to begin transaction"))

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.AddWithTransaction(context.Background(), entity)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	assert.Equal(t, "failed to begin transaction: failed to begin transaction", err.Error())
	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_InsertError(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	entity := &Entity{
		Name:       "Test User",
		Email:      "test@example.com",
		Position:   "Engineer",
		Department: "Engineering",
		RoleId:     1,
	}

	mockRepo.On("BeginTransaction", mock.Anything).Return((*sqlx.Tx)(nil), errors.New("insert failed"))

	svc := NewService(mockRepo, validator, logger)

	result, err := svc.AddWithTransaction(context.Background(), entity)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	assert.Contains(t, err.Error(), "failed to begin transaction: insert failed")
	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_Success(t *testing.T) {
	// Мок базы данных
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)

	defer func() {
		_ = db.Close() // игнорируем ошибку закрытия в тесте
	}()

	// sqlx.DB из обычного sql.DB
	sqlxDB := sqlx.NewDb(db, "postgres")

	sqlMock.ExpectBegin()

	// Запрос для проверки существования сотрудника --> возвращает пустой результат
	sqlMock.ExpectQuery(`SELECT id FROM employee WHERE name = \$1`).
		WithArgs("Jack Black").
		WillReturnError(sql.ErrNoRows)

	// INSERT запрос с возвратом ID
	sqlMock.ExpectQuery(`INSERT INTO employee \(name, email, position, department, role_id\) VALUES \(\$1, \$2, \$3, \$4, \$5\) RETURNING id`).
		WithArgs("Jack Black", "jack.black@example.com", "Developer", "IT", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(123))

	sqlMock.ExpectCommit()

	repo := NewEmployeeRepository(sqlxDB)
	validator := new(MockValidator)
	logger := createTestLogger()

	service := NewService(repo, validator, logger)

	employee := &Entity{
		Name:       "Jack Black",
		Email:      "jack.black@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     2,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	response, err := service.AddWithTransaction(context.Background(), employee)

	assert.NoError(t, err)
	assert.Equal(t, int64(123), response.Id)
	assert.Equal(t, "Jack Black", response.Name)
	assert.Equal(t, "jack.black@example.com", response.Email)
	assert.Equal(t, "Developer", response.Position)
	assert.Equal(t, "IT", response.Department)
	assert.Equal(t, int64(2), response.RoleId)

	assert.Equal(t, int64(123), employee.Id)

	// Проверка, что все ожидания были выполнены
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestCreateEmployee_Success(t *testing.T) {
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
	sqlMock.ExpectCommit()

	tx, err := sqlxDB.Beginx()
	assert.NoError(t, err)

	service := NewService(mockRepo, mockValidator, logger)

	employee := &Entity{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     2,
	}

	request := CreateRequest{
		Name:       employee.Name,
		Email:      employee.Email,
		Position:   employee.Position,
		Department: employee.Department,
		RoleId:     employee.RoleId,
	}

	expectedId := int64(123)

	mockValidator.On("Validate", request).Return(nil)
	mockRepo.On("BeginTransaction", mock.Anything).Return(tx, nil)
	mockRepo.On("FindByNameTx", mock.Anything, tx, "John Doe").Return(false, nil)
	mockRepo.On("SaveTx", mock.Anything, tx, request.ToEntity()).Return(expectedId, nil)

	result, err := service.CreateEmployee(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, expectedId, result)
	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestCreateEmployee_ValidationError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockValidator := new(MockValidator)
	logger := createTestLogger()

	service := NewService(mockRepo, mockValidator, logger)
	request := CreateRequest{Name: "A"} // слишком короткое имя

	validationErr := validator.ValidationErrors{}
	mockValidator.On("Validate", request).Return(validationErr)

	result, err := service.CreateEmployee(context.Background(), request)

	assert.Error(t, err)
	assert.Equal(t, int64(0), result)

	var reqValidationErr common.RequestValidationError
	assert.True(t, errors.As(err, &reqValidationErr))

	mockValidator.AssertExpectations(t)
	// Репозиторий не должен вызываться при ошибке валидации
	mockRepo.AssertNotCalled(t, "BeginTransaction")
}

func TestCreateEmployee_TransactionCreationError(t *testing.T) {
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

	employee := &Entity{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     2,
	}

	request := CreateRequest{
		Name:       employee.Name,
		Email:      employee.Email,
		Position:   employee.Position,
		Department: employee.Department,
		RoleId:     employee.RoleId,
	}

	txErr := errors.New("insert failed")

	mockValidator.On("Validate", request).Return(nil)
	mockRepo.On("BeginTransaction", mock.Anything).Return(tx, txErr)

	service := NewService(mockRepo, mockValidator, logger)

	result, err := service.CreateEmployee(context.Background(), request)

	assert.Error(t, err)
	assert.Equal(t, int64(0), result)
	assert.Contains(t, err.Error(), "error create employee: error creating transaction: insert failed")
	assert.Contains(t, err.Error(), txErr.Error())

	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCreateEmployee_FindByNameError(t *testing.T) {
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

	employee := &Entity{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     2,
	}

	request := CreateRequest{
		Name:       employee.Name,
		Email:      employee.Email,
		Position:   employee.Position,
		Department: employee.Department,
		RoleId:     employee.RoleId,
	}

	findErr := errors.New("database error")

	mockValidator.On("Validate", request).Return(nil)
	mockRepo.On("BeginTransaction", mock.Anything).Return(tx, nil)
	mockRepo.On("FindByNameTx", mock.Anything, tx, "John Doe").Return(false, findErr)

	result, err := service.CreateEmployee(context.Background(), request)

	assert.Error(t, err)
	assert.Equal(t, int64(0), result)
	assert.Contains(t, err.Error(), "error finding employee by name: John Doe")
	assert.Contains(t, err.Error(), findErr.Error())

	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestCreateEmployee_EmployeeAlreadyExists(t *testing.T) {
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

	employee := &Entity{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     2,
	}

	request := CreateRequest{
		Name:       employee.Name,
		Email:      employee.Email,
		Position:   employee.Position,
		Department: employee.Department,
		RoleId:     employee.RoleId,
	}

	mockValidator.On("Validate", request).Return(nil)
	mockRepo.On("BeginTransaction", mock.Anything).Return(tx, nil)
	mockRepo.On("FindByNameTx", mock.Anything, tx, "John Doe").Return(true, nil) // сотрудник существует

	result, err := service.CreateEmployee(context.Background(), request)

	assert.Error(t, err)
	assert.Equal(t, int64(0), result)

	var alreadyExistsErr common.AlreadyExistsError
	assert.True(t, errors.As(err, &alreadyExistsErr))
	assert.Contains(t, err.Error(), "employee with name John Doe already exists")

	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	// SaveTx не должен вызываться, если сотрудник уже существует
	mockRepo.AssertNotCalled(t, "SaveTx")
	assert.Error(t, sqlMock.ExpectationsWereMet())
}

func TestCreateEmployee_SaveError(t *testing.T) {
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

	employee := &Entity{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     2,
	}

	request := CreateRequest{
		Name:       employee.Name,
		Email:      employee.Email,
		Position:   employee.Position,
		Department: employee.Department,
		RoleId:     employee.RoleId,
	}

	saveErr := errors.New("save failed")

	mockValidator.On("Validate", request).Return(nil)
	mockRepo.On("BeginTransaction", mock.Anything).Return(tx, nil)
	mockRepo.On("FindByNameTx", mock.Anything, tx, "John Doe").Return(false, nil)
	mockRepo.On("SaveTx", mock.Anything, tx, request.ToEntity()).Return(int64(0), saveErr)

	result, err := service.CreateEmployee(context.Background(), request)

	assert.Error(t, err)
	assert.Equal(t, int64(0), result)
	assert.Contains(t, err.Error(), "error creating employee with name: John Doe")

	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

// Бенчмарк тест для проверки производительности
func BenchmarkCreateEmployee_Success(b *testing.B) {
	mockRepo := new(MockRepo)
	mockValidator := new(MockValidator)
	logger := createTestLogger()

	db, sqlMock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "postgres")

	service := NewService(mockRepo, mockValidator, logger)

	employee := &Entity{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     2,
	}

	request := CreateRequest{
		Name:       employee.Name,
		Email:      employee.Email,
		Position:   employee.Position,
		Department: employee.Department,
		RoleId:     employee.RoleId,
	}

	for b.Loop() {
		sqlMock.ExpectBegin()
		sqlMock.ExpectCommit()

		tx, _ := sqlxDB.Beginx()

		mockValidator.On("Validate", request).Return(nil).Once()
		mockRepo.On("BeginTransaction", mock.Anything).Return(tx, nil).Once()
		mockRepo.On("FindByNameTx", mock.Anything, tx, "John Doe").Return(false, nil).Once()
		mockRepo.On("SaveTx", mock.Anything, tx, request.ToEntity()).Return(int64(123), nil).Once()

		_, _ = service.CreateEmployee(context.Background(), request)
	}
}

func TestService_validateCreateRequest_Success(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	logger := createTestLogger()
	svc := NewService(mockRepo, validator, logger)

	request := CreateRequest{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     1,
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
