package employee

import (
	"database/sql"
	"errors"
	"idm/inner/common"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func (m *MockRepo) FindById(id int64) (Entity, error) {
	args := m.Called(id)
	return args.Get(0).(Entity), args.Error(1)
}

func (s *StubRepo) FindById(id int64) (Entity, error) {
	return s.entity, nil
}

func (m *MockRepo) Add(employee *Entity) error {
	args := m.Called(employee)
	return args.Error(0)
}

func (s *StubRepo) Add(entity *Entity) error {
	return nil
}

func (m *MockRepo) AddWithTransaction(tx *sqlx.Tx, employee *Entity) error {
	return m.Called(tx, employee).Error(0)
}

func (m *MockRepo) BeginTransaction() (*sqlx.Tx, error) {
	args := m.Called()
	var tx *sqlx.Tx
	if val := args.Get(0); val != nil {
		tx = val.(*sqlx.Tx)
	}
	return tx, args.Error(1)
}

func (s *StubRepo) AddWithTransaction(tx *sqlx.Tx, employee *Entity) error {
	return nil // или return errors.New("some error") для тестирования падающих кейсов
}

func (s *StubRepo) BeginTransaction() (*sqlx.Tx, error) {
	return &sqlx.Tx{}, nil // или return nil, errors.New("failed to begin transaction") Если нужно проверить логику при ошибке открытия транзакции
}

func (m *MockRepo) FindAll() ([]Entity, error) {
	args := m.Called()
	return args.Get(0).([]Entity), args.Error(1)
}

func (s *StubRepo) FindAll() ([]Entity, error) {
	return []Entity{s.entity}, nil
}

func (m *MockRepo) FindByIds(ids []int64) ([]Entity, error) {
	args := m.Called(ids)
	return args.Get(0).([]Entity), args.Error(1)
}

func (s *StubRepo) FindByIds(id []int64) ([]Entity, error) {
	return []Entity{s.entity}, nil
}

func (m *MockRepo) DeleteById(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (s *StubRepo) DeleteById(id int64) error {
	return nil
}

func (m *MockRepo) DeleteByIds(ids []int64) error {
	args := m.Called(ids)
	return args.Error(0)
}

func (s *StubRepo) DeleteByIds(id []int64) error {
	return nil
}

func (m *MockRepo) FindByNameTx(tx *sqlx.Tx, name string) (bool, error) {
	args := m.Called(tx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepo) SaveTx(tx *sqlx.Tx, employee Entity) (int64, error) {
	args := m.Called(tx, employee)
	return args.Get(0).(int64), args.Error(1)
}

func (s *StubRepo) FindByNameTx(tx *sqlx.Tx, name string) (bool, error) {
	panic("unimplemented")
}

func (s *StubRepo) SaveTx(tx *sqlx.Tx, employee Entity) (int64, error) {
	panic("unimplemented")
}

func TestService_FindById_Mock(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
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
	mockRepo.On("FindById", int64(1)).Return(entity, nil)

	svc := NewService(mockRepo, validator)

	result, err := svc.FindById(1)

	assert.NoError(t, err)
	assert.Equal(t, entity.toResponse(), result)
	mockRepo.AssertExpectations(t)
}

func TestService_FindById_Stub(t *testing.T) {
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

	svc := NewService(stubRepo, validator)

	result, err := svc.FindById(1)

	assert.NoError(t, err)
	assert.Equal(t, stubRepo.entity.toResponse(), result)
}

func TestService_FindById_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	mockRepo.On("FindById", int64(1)).Return(Entity{}, errors.New("db error"))

	svc := NewService(mockRepo, validator)

	result, err := svc.FindById(1)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	mockRepo.AssertExpectations(t)
}

func TestService_Add(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	entity := &Entity{
		Name:       "Jane",
		Email:      "jane@example.com",
		Position:   "Designer",
		Department: "Design",
		RoleId:     3,
	}
	mockRepo.On("Add", entity).Return(nil)

	svc := NewService(mockRepo, validator)

	result, err := svc.Add(entity)

	assert.NoError(t, err)
	assert.Equal(t, entity.toResponse(), result)
	mockRepo.AssertExpectations(t)
}

func TestService_Add_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	mockRepo.On("Add", mock.Anything).Return(errors.New("db error"))

	svc := NewService(mockRepo, validator)

	result, err := svc.Add(&Entity{})

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	mockRepo.AssertExpectations(t)
}

func TestService_FindAll(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
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
	mockRepo.On("FindAll").Return(entities, nil)

	svc := NewService(mockRepo, validator)

	result, err := svc.FindAll()

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, entities[0].toResponse(), result[0])
	assert.Equal(t, entities[1].toResponse(), result[1])
	mockRepo.AssertExpectations(t)
}

func TestService_FindAll_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	mockRepo.On("FindAll").Return([]Entity{}, errors.New("db error"))

	svc := NewService(mockRepo, validator)

	result, err := svc.FindAll()

	assert.Error(t, err)
	assert.Equal(t, 0, len(result))
	mockRepo.AssertExpectations(t)
}

func TestService_FindByIds(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
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
	mockRepo.On("FindByIds", ids).Return(entities, nil)

	svc := NewService(mockRepo, validator)

	result, err := svc.FindByIds(ids)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, entities[0].toResponse(), result[0])
	assert.Equal(t, entities[1].toResponse(), result[1])
	mockRepo.AssertExpectations(t)
}

func TestService_FindByIds_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	mockRepo.On("FindByIds", []int64{1, 2}).Return([]Entity{}, errors.New("db error"))

	svc := NewService(mockRepo, validator)

	result, err := svc.FindByIds([]int64{1, 2})

	assert.Error(t, err)
	assert.Equal(t, 0, len(result))
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteById(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	mockRepo.On("DeleteById", int64(1)).Return(nil)

	svc := NewService(mockRepo, validator)

	err := svc.DeleteById(1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteById_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	mockRepo.On("DeleteById", int64(1)).Return(errors.New("db error"))

	svc := NewService(mockRepo, validator)

	err := svc.DeleteById(1)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteByIds(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	mockRepo.On("DeleteByIds", []int64{1, 2}).Return(nil)

	svc := NewService(mockRepo, validator)

	err := svc.DeleteByIds([]int64{1, 2})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteByIds_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	mockRepo.On("DeleteByIds", []int64{1, 2}).Return(errors.New("db error"))

	svc := NewService(mockRepo, validator)

	err := svc.DeleteByIds([]int64{1, 2})

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_BeginError(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	mockRepo.On("BeginTransaction").Return(nil, errors.New("failed to begin transaction"))

	svc := NewService(mockRepo, validator)

	entity := &Entity{
		Name:       "Test User",
		Email:      "test@example.com",
		Position:   "Engineer",
		Department: "Engineering",
		RoleId:     1,
	}

	result, err := svc.AddWithTransaction(entity)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	assert.Contains(t, err.Error(), "failed to begin")

	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_ExistingEmployeeCheckError(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	entity := &Entity{
		Name:       "Test User",
		Email:      "test@example.com",
		Position:   "Engineer",
		Department: "Engineering",
		RoleId:     1,
	}

	mockRepo.On("BeginTransaction").Return((*sqlx.Tx)(nil), errors.New("transaction failed"))

	svc := NewService(mockRepo, validator)

	result, err := svc.AddWithTransaction(entity)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	assert.Contains(t, err.Error(), "transaction failed")
	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_EmployeeAlreadyExists(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	entity := &Entity{
		Name:       "Test User",
		Email:      "test@example.com",
		Position:   "Engineer",
		Department: "Engineering",
		RoleId:     1,
	}

	mockRepo.On("BeginTransaction").Return((*sqlx.Tx)(nil), errors.New("failed to begin transaction"))

	svc := NewService(mockRepo, validator)

	result, err := svc.AddWithTransaction(entity)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	assert.Equal(t, "failed to begin transaction: failed to begin transaction", err.Error())
	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_InsertError(t *testing.T) {
	mockRepo := new(MockRepo)
	validator := new(MockValidator)
	entity := &Entity{
		Name:       "Test User",
		Email:      "test@example.com",
		Position:   "Engineer",
		Department: "Engineering",
		RoleId:     1,
	}

	mockRepo.On("BeginTransaction").Return((*sqlx.Tx)(nil), errors.New("insert failed"))

	svc := NewService(mockRepo, validator)

	result, err := svc.AddWithTransaction(entity)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	assert.Contains(t, err.Error(), "failed to begin transaction: insert failed")
	mockRepo.AssertExpectations(t)
}

func TestService_AddWithTransaction_Success(t *testing.T) {
	// Мок базы данных
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	defer func() {
		_ = db.Close() // игнорируем ошибку закрытия в тесте
	}()

	// sqlx.DB из обычного sql.DB
	sqlxDB := sqlx.NewDb(db, "postgres")

	mock.ExpectBegin()

	// Запрос для проверки существования сотрудника --> возвращает пустой результат
	mock.ExpectQuery(`SELECT id FROM employee WHERE name = \$1`).
		WithArgs("Jack Black").
		WillReturnError(sql.ErrNoRows)

	// INSERT запрос с возвратом ID
	mock.ExpectQuery(`INSERT INTO employee \(name, email, position, department, role_id\) VALUES \(\$1, \$2, \$3, \$4, \$5\) RETURNING id`).
		WithArgs("Jack Black", "jack.black@example.com", "Developer", "IT", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(123))

	mock.ExpectCommit()

	repo := NewEmployeeRepository(sqlxDB)
	validator := new(MockValidator)

	service := NewService(repo, validator)

	employee := &Entity{
		Name:       "Jack Black",
		Email:      "jack.black@example.com",
		Position:   "Developer",
		Department: "IT",
		RoleId:     2,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	response, err := service.AddWithTransaction(employee)

	assert.NoError(t, err)
	assert.Equal(t, int64(123), response.Id)
	assert.Equal(t, "Jack Black", response.Name)
	assert.Equal(t, "jack.black@example.com", response.Email)
	assert.Equal(t, "Developer", response.Position)
	assert.Equal(t, "IT", response.Department)
	assert.Equal(t, int64(2), response.RoleId)

	assert.Equal(t, int64(123), employee.Id)

	// Проверка, что все ожидания были выполнены
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateEmployee_Success(t *testing.T) {
	mockRepo := new(MockRepo)
	mockValidator := new(MockValidator)

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "postgres")
	mock.ExpectBegin()
	mock.ExpectCommit()

	tx, err := sqlxDB.Beginx()
	assert.NoError(t, err)

	service := NewService(mockRepo, mockValidator)

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
	mockRepo.On("BeginTransaction").Return(tx, nil)
	mockRepo.On("FindByNameTx", tx, "John Doe").Return(false, nil)
	mockRepo.On("SaveTx", tx, request.ToEntity()).Return(expectedId, nil)

	result, err := service.CreateEmployee(request)

	assert.NoError(t, err)
	assert.Equal(t, expectedId, result)
	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateEmployee_ValidationError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockValidator := new(MockValidator)

	service := NewService(mockRepo, mockValidator)
	request := CreateRequest{Name: "A"} // слишком короткое имя

	validationErr := validator.ValidationErrors{}
	mockValidator.On("Validate", request).Return(validationErr)

	result, err := service.CreateEmployee(request)

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

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "postgres")
	mock.ExpectBegin()
	mock.ExpectRollback()

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
	mockRepo.On("BeginTransaction").Return(tx, txErr)

	service := NewService(mockRepo, mockValidator)

	result, err := service.CreateEmployee(request)

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

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "postgres")
	mock.ExpectBegin()
	mock.ExpectRollback()

	tx, err := sqlxDB.Beginx()
	assert.NoError(t, err)

	service := NewService(mockRepo, mockValidator)

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
	mockRepo.On("BeginTransaction").Return(tx, nil)
	mockRepo.On("FindByNameTx", tx, "John Doe").Return(false, findErr)

	result, err := service.CreateEmployee(request)

	assert.Error(t, err)
	assert.Equal(t, int64(0), result)
	assert.Contains(t, err.Error(), "error finding employee by name: John Doe")
	assert.Contains(t, err.Error(), findErr.Error())

	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateEmployee_EmployeeAlreadyExists(t *testing.T) {
	mockRepo := new(MockRepo)
	mockValidator := new(MockValidator)

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "postgres")
	mock.ExpectBegin()
	mock.ExpectRollback()

	tx, err := sqlxDB.Beginx()
	assert.NoError(t, err)

	service := NewService(mockRepo, mockValidator)

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
	mockRepo.On("BeginTransaction").Return(tx, nil)
	mockRepo.On("FindByNameTx", tx, "John Doe").Return(true, nil) // сотрудник существует

	result, err := service.CreateEmployee(request)

	assert.Error(t, err)
	assert.Equal(t, int64(0), result)

	var alreadyExistsErr common.AlreadyExistsError
	assert.True(t, errors.As(err, &alreadyExistsErr))
	assert.Contains(t, err.Error(), "employee with name John Doe already exists")

	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	// SaveTx не должен вызываться, если сотрудник уже существует
	mockRepo.AssertNotCalled(t, "SaveTx")
	assert.Error(t, mock.ExpectationsWereMet())
}

func TestCreateEmployee_SaveError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockValidator := new(MockValidator)

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "postgres")
	mock.ExpectBegin()
	mock.ExpectRollback()

	tx, err := sqlxDB.Beginx()
	assert.NoError(t, err)

	service := NewService(mockRepo, mockValidator)

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
	mockRepo.On("BeginTransaction").Return(tx, nil)
	mockRepo.On("FindByNameTx", tx, "John Doe").Return(false, nil)
	mockRepo.On("SaveTx", tx, request.ToEntity()).Return(int64(0), saveErr)

	result, err := service.CreateEmployee(request)

	assert.Error(t, err)
	assert.Equal(t, int64(0), result)
	assert.Contains(t, err.Error(), "error creating employee with name: John Doe")

	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Бенчмарк тест для проверки производительности
func BenchmarkCreateEmployee_Success(b *testing.B) {
	mockRepo := new(MockRepo)
	mockValidator := new(MockValidator)

	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "postgres")

	service := NewService(mockRepo, mockValidator)

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
		mock.ExpectBegin()
		mock.ExpectCommit()

		tx, _ := sqlxDB.Beginx()

		mockValidator.On("Validate", request).Return(nil).Once()
		mockRepo.On("BeginTransaction").Return(tx, nil).Once()
		mockRepo.On("FindByNameTx", tx, "John Doe").Return(false, nil).Once()
		mockRepo.On("SaveTx", tx, request.ToEntity()).Return(int64(123), nil).Once()

		_, _ = service.CreateEmployee(request)
	}
}
