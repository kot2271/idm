package role

import (
	"fmt"
)

type Service struct {
	repo Repo
}

type Repo interface {
	FindById(id int64) (Entity, error)
	Add(role *Entity) error
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
	var role, err = svc.repo.FindById(id)
	if err != nil {
		return Response{}, fmt.Errorf("error finding role with id %d: %w", id, err)
	}

	return role.toResponse(), nil
}

func (svc *Service) Add(role *Entity) (Response, error) {
	err := svc.repo.Add(role)
	if err != nil {
		return Response{}, fmt.Errorf("error adding role: %w", err)
	}
	return role.toResponse(), nil
}

func (svc *Service) FindAll() ([]Response, error) {
	roles, err := svc.repo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("error finding all roles: %w", err)
	}
	var responses []Response
	for _, role := range roles {
		responses = append(responses, role.toResponse())
	}
	return responses, nil
}

func (svc *Service) FindByIds(ids []int64) ([]Response, error) {
	roles, err := svc.repo.FindByIds(ids)
	if err != nil {
		return nil, fmt.Errorf("error finding roles by ids: %w", err)
	}
	var responses []Response
	for _, role := range roles {
		responses = append(responses, role.toResponse())
	}
	return responses, nil
}

func (svc *Service) DeleteById(id int64) error {
	err := svc.repo.DeleteById(id)
	if err != nil {
		return fmt.Errorf("error deleting role with id %d: %w", id, err)
	}
	return nil
}

func (svc *Service) DeleteByIds(ids []int64) error {
	err := svc.repo.DeleteByIds(ids)
	if err != nil {
		return fmt.Errorf("error deleting roles with ids: %w", err)
	}
	return nil
}
