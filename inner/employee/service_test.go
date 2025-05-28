package employee

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// объявляем структуру мок-репозитория
type MockRepo struct {
	mock.Mock
}

// объявляем структуру Stub-репозитория
type StubRepo struct {
	entity Entity
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

func TestService_FindById_Mock(t *testing.T) {
	mockRepo := new(MockRepo)
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

	svc := NewService(mockRepo)

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

	svc := NewService(stubRepo)

	result, err := svc.FindById(1)

	assert.NoError(t, err)
	assert.Equal(t, stubRepo.entity.toResponse(), result)
}

func TestService_FindById_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	mockRepo.On("FindById", int64(1)).Return(Entity{}, errors.New("db error"))

	svc := NewService(mockRepo)

	result, err := svc.FindById(1)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	mockRepo.AssertExpectations(t)
}

func TestService_Add(t *testing.T) {
	mockRepo := new(MockRepo)
	entity := &Entity{
		Name:       "Jane",
		Email:      "jane@example.com",
		Position:   "Designer",
		Department: "Design",
		RoleId:     3,
	}
	mockRepo.On("Add", entity).Return(nil)

	svc := NewService(mockRepo)

	result, err := svc.Add(entity)

	assert.NoError(t, err)
	assert.Equal(t, entity.toResponse(), result)
	mockRepo.AssertExpectations(t)
}

func TestService_Add_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	mockRepo.On("Add", mock.Anything).Return(errors.New("db error"))

	svc := NewService(mockRepo)

	result, err := svc.Add(&Entity{})

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	mockRepo.AssertExpectations(t)
}

func TestService_FindAll(t *testing.T) {
	mockRepo := new(MockRepo)
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

	svc := NewService(mockRepo)

	result, err := svc.FindAll()

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, entities[0].toResponse(), result[0])
	assert.Equal(t, entities[1].toResponse(), result[1])
	mockRepo.AssertExpectations(t)
}

func TestService_FindAll_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	mockRepo.On("FindAll").Return([]Entity{}, errors.New("db error"))

	svc := NewService(mockRepo)

	result, err := svc.FindAll()

	assert.Error(t, err)
	assert.Equal(t, 0, len(result))
	mockRepo.AssertExpectations(t)
}

func TestService_FindByIds(t *testing.T) {
	mockRepo := new(MockRepo)
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

	svc := NewService(mockRepo)

	result, err := svc.FindByIds(ids)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, entities[0].toResponse(), result[0])
	assert.Equal(t, entities[1].toResponse(), result[1])
	mockRepo.AssertExpectations(t)
}

func TestService_FindByIds_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	mockRepo.On("FindByIds", []int64{1, 2}).Return([]Entity{}, errors.New("db error"))

	svc := NewService(mockRepo)

	result, err := svc.FindByIds([]int64{1, 2})

	assert.Error(t, err)
	assert.Equal(t, 0, len(result))
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteById(t *testing.T) {
	mockRepo := new(MockRepo)
	mockRepo.On("DeleteById", int64(1)).Return(nil)

	svc := NewService(mockRepo)

	err := svc.DeleteById(1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteById_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	mockRepo.On("DeleteById", int64(1)).Return(errors.New("db error"))

	svc := NewService(mockRepo)

	err := svc.DeleteById(1)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteByIds(t *testing.T) {
	mockRepo := new(MockRepo)
	mockRepo.On("DeleteByIds", []int64{1, 2}).Return(nil)

	svc := NewService(mockRepo)

	err := svc.DeleteByIds([]int64{1, 2})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteByIds_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	mockRepo.On("DeleteByIds", []int64{1, 2}).Return(errors.New("db error"))

	svc := NewService(mockRepo)

	err := svc.DeleteByIds([]int64{1, 2})

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}
