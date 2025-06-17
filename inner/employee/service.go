package employee

import (
	"fmt"

	"idm/inner/common"
	"idm/inner/validator"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Service struct {
	repo      Repo
	validator Validator
	logger    *common.Logger
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
func NewService(repo Repo, validator Validator, logger *common.Logger) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
		logger:    logger,
	}
}

// Метод для создания нового сотрудника
// принимает на вход CreateRequest - структура запроса на создание сотрудника
func (svc *Service) CreateEmployee(request CreateRequest) (int64, error) {
	svc.logger.Info("Creating new employee", zap.String("name", request.Name))

	if err := svc.validateCreateRequest(request); err != nil {
		return 0, err
	}

	// запрашиваем у репозитория новую транзакцию
	tx, err := svc.repo.BeginTransaction()
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				svc.logger.Error("Failed to rollback transaction",
					zap.String("name", request.Name),
					zap.Error(rollbackErr))
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				svc.logger.Error("Failed to commit transaction",
					zap.String("name", request.Name),
					zap.Error(commitErr))
				err = commitErr
			}
		}
	}()
	if err != nil {
		svc.logger.Error("Failed to begin transaction for employee creation",
			zap.String("name", request.Name),
			zap.Error(err))
		return 0, fmt.Errorf("error create employee: error creating transaction: %w", err)
	}

	// в рамках транзакции проверяем наличие в базе данных работника с таким же именем
	isExist, err := svc.repo.FindByNameTx(tx, request.Name)
	if err != nil {
		svc.logger.Error("Failed to check if employee exists",
			zap.String("name", request.Name),
			zap.Error(err))
		return 0, fmt.Errorf("error finding employee by name: %s, %w", request.Name, err)
	}
	if isExist {
		svc.logger.Warn("Employee with this name already exists",
			zap.String("name", request.Name))
		return 0, common.AlreadyExistsError{Message: fmt.Sprintf("employee with name %s already exists", request.Name)}
	}

	// в случае отсутствия сотрудника с таким же именем - в рамках этой же транзакции вызываем метод репозитория,
	// который должен будет создать нового сотрудника
	newEmployeeId, err := svc.repo.SaveTx(tx, request.ToEntity())
	if err != nil {
		svc.logger.Error("Failed to save new employee",
			zap.String("name", request.Name),
			zap.Error(err))
		err = fmt.Errorf("error creating employee with name: %s %v", request.Name, err)
	} else {
		svc.logger.Info("Employee created successfully",
			zap.String("name", request.Name),
			zap.Int64("id", newEmployeeId))
	}
	return newEmployeeId, err
}

// валидация запроса на создание сотрудника
func (svc *Service) validateCreateRequest(request CreateRequest) error {
	svc.logger.Debug("Validating create employee request", zap.Any("request", request))

	err := svc.validator.Validate(request)
	if err != nil {
		svc.logger.Error("Employee creation request validation failed",
			zap.String("name", request.Name),
			zap.Error(err))

		if validationErr, ok := err.(validator.ValidationErrors); ok {
			return common.RequestValidationError{
				Message: "Data validation error",
				Data:    validationErr.Errors,
			}
		}

		// Если это другая ошибка валидации, возвращаем её как есть
		return common.RequestValidationError{Message: err.Error()}
	}

	return nil
}

func (svc *Service) FindById(id int64) (Response, error) {
	svc.logger.Debug("Finding employee by ID", zap.Int64("id", id))

	var entity, err = svc.repo.FindById(id)
	if err != nil {
		svc.logger.Error("Failed to find employee by ID",
			zap.Int64("id", id),
			zap.Error(err))
		return Response{}, fmt.Errorf("error finding employee with id %d: %w", id, err)
	}

	svc.logger.Debug("Employee found successfully", zap.Int64("id", id))
	return entity.toResponse(), nil
}

func (svc *Service) Add(employee *Entity) (Response, error) {
	svc.logger.Info("Adding employee", zap.String("name", employee.Name))

	err := svc.validator.Validate(employee)
	if err != nil {
		svc.logger.Error("Employee add request validation failed",
			zap.String("name", employee.Name),
			zap.Error(err))

		if validationErr, ok := err.(validator.ValidationErrors); ok {
			return Response{}, common.RequestValidationError{
				Message: "Data validation error",
				Data:    validationErr.Errors,
			}
		}
		return Response{}, common.RequestValidationError{Message: err.Error()}
	}

	err = svc.repo.Add(employee)
	if err != nil {
		svc.logger.Error("Failed to add employee",
			zap.String("name", employee.Name),
			zap.Error(err))
		return Response{}, fmt.Errorf("error adding employee: %w", err)
	}
	svc.logger.Info("Employee added successfully", zap.String("name", employee.Name))
	return employee.toResponse(), nil
}

func (svc *Service) AddWithTransaction(employee *Entity) (Response, error) {
	svc.logger.Info("Adding employee with transaction", zap.String("name", employee.Name))

	tx, err := svc.repo.BeginTransaction()
	if err != nil {
		svc.logger.Error("Failed to begin transaction for employee addition",
			zap.String("name", employee.Name),
			zap.Error(err))
		return Response{}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				svc.logger.Error("Rollback after panic failed",
					zap.String("name", employee.Name),
					zap.Error(rollbackErr))
			}
			panic(r)
		}

		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				svc.logger.Error("Rollback on error failed",
					zap.String("name", employee.Name),
					zap.Error(rollbackErr))
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				svc.logger.Error("Commit failed",
					zap.String("name", employee.Name),
					zap.Error(commitErr))
				err = fmt.Errorf("commit failed: %w", commitErr)
			}
		}
	}()

	err = svc.repo.AddWithTransaction(tx, employee)
	if err != nil {
		svc.logger.Error("Transaction failed while adding employee",
			zap.String("name", employee.Name),
			zap.Error(err))
		return Response{}, fmt.Errorf("transaction failed: %w", err)
	}

	svc.logger.Info("Employee added with transaction successfully", zap.String("name", employee.Name))
	return employee.toResponse(), nil
}

func (svc *Service) FindAll() ([]Response, error) {
	svc.logger.Debug("Finding all employees")

	entities, err := svc.repo.FindAll()
	if err != nil {
		svc.logger.Error("Failed to find all employees", zap.Error(err))
		return nil, fmt.Errorf("error finding all employees: %w", err)
	}

	responses := make([]Response, len(entities))
	for i, entity := range entities {
		responses[i] = entity.toResponse()
	}
	svc.logger.Debug("Found all employees", zap.Int("count", len(responses)))
	return responses, nil
}

func (svc *Service) FindByIds(ids []int64) ([]Response, error) {
	svc.logger.Debug("Finding employees by IDs", zap.Int64s("ids", ids))

	entities, err := svc.repo.FindByIds(ids)
	if err != nil {
		svc.logger.Error("Failed to find employees by IDs",
			zap.Int64s("ids", ids),
			zap.Error(err))
		return nil, fmt.Errorf("error finding employees by ids: %w", err)
	}

	responses := make([]Response, len(entities))
	for i, entity := range entities {
		responses[i] = entity.toResponse()
	}
	svc.logger.Debug("Found employees by IDs",
		zap.Int64s("ids", ids),
		zap.Int("found_count", len(responses)))
	return responses, nil
}

func (svc *Service) DeleteById(id int64) error {
	svc.logger.Info("Deleting employee by ID", zap.Int64("id", id))

	err := svc.repo.DeleteById(id)
	if err != nil {
		svc.logger.Error("Failed to delete employee by ID",
			zap.Int64("id", id),
			zap.Error(err))
		return fmt.Errorf("error deleting employee with id %d: %w", id, err)
	}

	svc.logger.Info("Employee deleted successfully", zap.Int64("id", id))
	return nil
}

func (svc *Service) DeleteByIds(ids []int64) error {
	svc.logger.Info("Deleting employees by IDs", zap.Int64s("ids", ids))

	err := svc.repo.DeleteByIds(ids)
	if err != nil {
		svc.logger.Error("Failed to delete employees by IDs",
			zap.Int64s("ids", ids),
			zap.Error(err))
		return fmt.Errorf("error deleting employees with ids: %w", err)
	}

	svc.logger.Info("Employees deleted successfully", zap.Int64s("ids", ids))
	return nil
}
