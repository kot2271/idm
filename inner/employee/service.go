package employee

import (
	"fmt"
)

type Service struct {
	repo Repo
}

type Repo interface {
	FindById(id int64) (Entity, error)
	Add(employee *Entity) error
	FindAll() ([]Entity, error)
	FindByIds(ids []int64) ([]Entity, error)
	DeleteById(id int64) error
	DeleteByIds(ids []int64) error
}

// функция-конструктор
func NewService(repo Repo) *Service {
	return &Service{
		repo: repo,
	}
}

func (svc *Service) FindById(id int64) (Response, error) {
	var entity, err = svc.repo.FindById(id)
	if err != nil {
		return Response{}, fmt.Errorf("error finding employee with id %d: %w", id, err)
	}

	return entity.toResponse(), nil
}

func (svc *Service) Add(employee *Entity) (Response, error) {
	err := svc.repo.Add(employee)
	if err != nil {
		return Response{}, fmt.Errorf("error adding employee: %w", err)
	}
	return employee.toResponse(), nil
}

func (svc *Service) FindAll() ([]Response, error) {
	entities, err := svc.repo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("error finding all employees: %w", err)
	}
	var responses []Response
	for _, entity := range entities {
		responses = append(responses, entity.toResponse())
	}
	return responses, nil
}

func (svc *Service) FindByIds(ids []int64) ([]Response, error) {
	entities, err := svc.repo.FindByIds(ids)
	if err != nil {
		return nil, fmt.Errorf("error finding employees by ids: %w", err)
	}
	var responses []Response
	for _, entity := range entities {
		responses = append(responses, entity.toResponse())
	}
	return responses, nil
}

func (svc *Service) DeleteById(id int64) error {
	err := svc.repo.DeleteById(id)
	if err != nil {
		return fmt.Errorf("error deleting employee with id %d: %w", id, err)
	}
	return nil
}

func (svc *Service) DeleteByIds(ids []int64) error {
	err := svc.repo.DeleteByIds(ids)
	if err != nil {
		return fmt.Errorf("error deleting employees with ids: %w", err)
	}
	return nil
}
