package role

import (
	"fmt"

	"idm/inner/common"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	repo      Repo
	validator Validator
}

type Repo interface {
	FindById(id int64) (Entity, error)
	Add(role *Entity) error
	FindAll() ([]Entity, error)
	FindByIds(ids []int64) ([]Entity, error)
	DeleteById(id int64) error
	DeleteByIds(ids []int64) error
	BeginTransaction() (*sqlx.Tx, error)
	FindByNameTx(tx *sqlx.Tx, name string) (bool, error)
	SaveTx(tx *sqlx.Tx, role Entity) (int64, error)
}

type Validator interface {
	Validate(request any) error
}

// функция-конструктор
func NewService(repo Repo, validator Validator) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
	}
}

// Метод для создания новой роли
// принимает на вход CreateRequest - структура запроса на создание новой роли
func (svc *Service) CreateRole(request CreateRequest) (int64, error) {

	// валидируем запрос
	var err = svc.validator.Validate(request)
	if err != nil {
		// возвращаем кастомную ошибку в случае, если запрос не прошёл валидацию
		return 0, common.RequestValidationError{Message: err.Error()}
	}

	// запрашиваем у репозитория новую транзакцию
	tx, err := svc.repo.BeginTransaction()
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	if err != nil {
		return 0, fmt.Errorf("error create role: error creating transaction: %w", err)
	}

	// в рамках транзакции проверяем наличие в базе данных роли с таким же именем
	isExist, err := svc.repo.FindByNameTx(tx, request.Name)
	if err != nil {
		return 0, fmt.Errorf("error finding role by name: %s, %w", request.Name, err)
	}
	if isExist {
		return 0, common.AlreadyExistsError{Message: fmt.Sprintf("role with name %s already exists", request.Name)}
	}

	// в случае отсутствия роли с таким же именем - в рамках этой же транзакции вызываем метод репозитория,
	// который должен будет создать новую роль
	newRoleId, err := svc.repo.SaveTx(tx, request.ToEntity())
	if err != nil {
		err = fmt.Errorf("error creating role with name: %s %v", request.Name, err)
	}
	return newRoleId, err
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
