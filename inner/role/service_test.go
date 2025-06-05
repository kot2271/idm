package role

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Объявляем структуру мок-репозитория
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) FindById(id int64) (Entity, error) {
	args := m.Called(id)
	return args.Get(0).(Entity), args.Error(1)
}

func (m *MockRepo) Add(role *Entity) error {
	args := m.Called(role)
	return args.Error(0)
}

func (m *MockRepo) FindAll() ([]Entity, error) {
	args := m.Called()
	return args.Get(0).([]Entity), args.Error(1)
}

func (m *MockRepo) FindByIds(ids []int64) ([]Entity, error) {
	args := m.Called(ids)
	return args.Get(0).([]Entity), args.Error(1)
}

func (m *MockRepo) DeleteById(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockRepo) DeleteByIds(ids []int64) error {
	args := m.Called(ids)
	return args.Error(0)
}

func TestService_FindById(t *testing.T) {
	mockRepo := new(MockRepo)
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

	svc := NewService(mockRepo)

	result, err := svc.FindById(2)

	assert.NoError(t, err)
	assert.Equal(t, entity.toResponse(), result)
	mockRepo.AssertExpectations(t)
}

func TestService_FindById_Error(t *testing.T) {
	mockRepo := new(MockRepo)
	mockRepo.On("FindById", int64(2)).Return(Entity{}, errors.New("db error"))

	svc := NewService(mockRepo)

	result, err := svc.FindById(2)

	assert.Error(t, err)
	assert.Equal(t, Response{}, result)
	mockRepo.AssertExpectations(t)
}

func TestService_Add(t *testing.T) {
	mockRepo := new(MockRepo)
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
	mockRepo.On("DeleteById", int64(2)).Return(nil)

	svc := NewService(mockRepo)

	err := svc.DeleteById(2)

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
	mockRepo.On("DeleteByIds", []int64{2, 3}).Return(nil)

	svc := NewService(mockRepo)

	err := svc.DeleteByIds([]int64{2, 3})

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
