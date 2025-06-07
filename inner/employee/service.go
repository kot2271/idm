package employee

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
	Add(employee *Entity) error
	AddWithTransaction(tx *sqlx.Tx, employee *Entity) error
	FindAll() ([]Entity, error)
	FindByIds(ids []int64) ([]Entity, error)
	DeleteById(id int64) error
	DeleteByIds(ids []int64) error
	BeginTransaction() (*sqlx.Tx, error)
	FindByNameTx(tx *sqlx.Tx, name string) (bool, error)
	SaveTx(tx *sqlx.Tx, employee Entity) (int64, error)
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

// Метод для создания нового сотрудника
// принимает на вход CreateRequest - структура запроса на создание сотрудника
func (svc *Service) CreateEmployee(request CreateRequest) (int64, error) {

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
		return 0, fmt.Errorf("error create employee: error creating transaction: %w", err)
	}

	// в рамках транзакции проверяем наличие в базе данных работника с таким же именем
	isExist, err := svc.repo.FindByNameTx(tx, request.Name)
	if err != nil {
		return 0, fmt.Errorf("error finding employee by name: %s, %w", request.Name, err)
	}
	if isExist {
		return 0, common.AlreadyExistsError{Message: fmt.Sprintf("employee with name %s already exists", request.Name)}
	}

	// в случае отсутствия сотрудника с таким же именем - в рамках этой же транзакции вызываем метод репозитория,
	// который должен будет создать нового сотрудника
	newEmployeeId, err := svc.repo.SaveTx(tx, request.ToEntity())
	if err != nil {
		err = fmt.Errorf("error creating employee with name: %s %v", request.Name, err)
	}
	return newEmployeeId, err
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

func (svc *Service) AddWithTransaction(employee *Entity) (Response, error) {
	tx, err := svc.repo.BeginTransaction()
	if err != nil {
		return Response{}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				fmt.Printf("rollback after panic failed: %v\n", rollbackErr)
			}
			panic(r)
		}

		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				fmt.Printf("rollback on error failed: %v\n", rollbackErr)
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				err = fmt.Errorf("commit failed: %w", commitErr)
			}
		}
	}()

	err = svc.repo.AddWithTransaction(tx, employee)
	if err != nil {
		return Response{}, fmt.Errorf("transaction failed: %w", err)
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
