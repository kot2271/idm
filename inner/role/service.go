package role

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
func NewService(repo Repo, validator Validator, logger *common.Logger) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
		logger:    logger,
	}
}

// Метод для создания новой роли
// принимает на вход CreateRequest - структура запроса на создание новой роли
func (svc *Service) CreateRole(request CreateRequest) (int64, error) {
	svc.logger.Info("Creating new role", zap.String("name", request.Name))

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
		svc.logger.Error("Failed to begin transaction for role creation",
			zap.String("name", request.Name),
			zap.Error(err))
		return 0, fmt.Errorf("error create role: error creating transaction: %w", err)
	}

	// в рамках транзакции проверяем наличие в базе данных роли с таким же именем
	isExist, err := svc.repo.FindByNameTx(tx, request.Name)
	if err != nil {
		svc.logger.Error("Failed to check role existence",
			zap.String("name", request.Name),
			zap.Error(err))
		return 0, fmt.Errorf("error finding role by name: %s, %w", request.Name, err)
	}
	if isExist {
		svc.logger.Warn("Role already exists",
			zap.String("name", request.Name))
		return 0, common.AlreadyExistsError{Message: fmt.Sprintf("role with name %s already exists", request.Name)}
	}

	// в случае отсутствия роли с таким же именем - в рамках этой же транзакции вызываем метод репозитория,
	// который должен будет создать новую роль
	newRoleId, err := svc.repo.SaveTx(tx, request.ToEntity())
	if err != nil {
		svc.logger.Error("Failed to save role",
			zap.String("name", request.Name),
			zap.Error(err))
		err = fmt.Errorf("error creating role with name: %s %v", request.Name, err)
	} else {
		svc.logger.Info("Role created successfully",
			zap.String("name", request.Name),
			zap.Int64("id", newRoleId))
	}

	return newRoleId, err
}

// валидация запроса на создание роли
func (svc *Service) validateCreateRequest(request CreateRequest) error {
	svc.logger.Debug("Validating create role request", zap.Any("request", request))

	err := svc.validator.Validate(request)
	if err != nil {
		svc.logger.Error("Role creation request validation failed",
			zap.String("name", request.Name),
			zap.Error(err))

		if validationErr, ok := err.(validator.ValidationErrors); ok {
			return common.RequestValidationError{
				Message: "Data validation error",
				Data:    validationErr.Errors,
			}
		}

		return common.RequestValidationError{Message: err.Error()}
	}

	return nil
}

func (svc *Service) FindById(id int64) (Response, error) {
	svc.logger.Debug("Finding role by ID", zap.Int64("id", id))

	var role, err = svc.repo.FindById(id)
	if err != nil {
		svc.logger.Error("Failed to find role by ID",
			zap.Int64("id", id),
			zap.Error(err))
		return Response{}, fmt.Errorf("error finding role with id %d: %w", id, err)
	}

	svc.logger.Debug("Role found successfully", zap.Int64("id", id))
	return role.toResponse(), nil
}

func (svc *Service) Add(role *Entity) (Response, error) {
	svc.logger.Info("Adding role", zap.String("name", role.Name))

	err := svc.validator.Validate(role)
	if err != nil {
		svc.logger.Error("Role add request validation failed",
			zap.String("name", role.Name),
			zap.Error(err))

		if validationErr, ok := err.(validator.ValidationErrors); ok {
			return Response{}, common.RequestValidationError{
				Message: "Data validation error",
				Data:    validationErr.Errors,
			}
		}
		return Response{}, common.RequestValidationError{Message: err.Error()}
	}

	err = svc.repo.Add(role)
	if err != nil {
		svc.logger.Error("Failed to add role",
			zap.String("name", role.Name),
			zap.Error(err))
		return Response{}, fmt.Errorf("error adding role: %w", err)
	}

	svc.logger.Info("Role added successfully", zap.String("name", role.Name))
	return role.toResponse(), nil
}

func (svc *Service) FindAll() ([]Response, error) {
	svc.logger.Debug("Fetching all roles")

	roles, err := svc.repo.FindAll()
	if err != nil {
		svc.logger.Error("Failed to fetch all roles", zap.Error(err))
		return nil, fmt.Errorf("error finding all roles: %w", err)
	}

	responses := make([]Response, len(roles))
	for i, entity := range roles {
		responses[i] = entity.toResponse()
	}

	svc.logger.Debug("Found all roles successfully", zap.Int("count", len(responses)))
	return responses, nil
}

func (svc *Service) FindByIds(ids []int64) ([]Response, error) {
	svc.logger.Debug("Finding roles by IDs", zap.Int64s("ids", ids))

	if len(ids) == 0 {
		svc.logger.Warn("No IDs provided for role search")
		return []Response{}, nil
	}

	roles, err := svc.repo.FindByIds(ids)
	if err != nil {
		svc.logger.Error("Failed to find roles by IDs",
			zap.Int64s("ids", ids),
			zap.Error(err))
		return nil, fmt.Errorf("error finding roles by ids: %w", err)
	}

	responses := make([]Response, len(roles))
	for i, entity := range roles {
		responses[i] = entity.toResponse()
	}

	svc.logger.Debug("Found roles by IDs successfully",
		zap.Int64s("ids", ids),
		zap.Int("found_count", len(responses)))
	return responses, nil
}

func (svc *Service) DeleteById(id int64) error {
	svc.logger.Info("Deleting role by ID", zap.Int64("id", id))

	err := svc.repo.DeleteById(id)
	if err != nil {
		svc.logger.Error("Failed to delete role by ID",
			zap.Int64("id", id),
			zap.Error(err))
		return fmt.Errorf("error deleting role with id %d: %w", id, err)
	}
	svc.logger.Info("Role deleted successfully", zap.Int64("id", id))
	return nil
}

func (svc *Service) DeleteByIds(ids []int64) error {
	svc.logger.Info("Deleting roles by IDs", zap.Int64s("ids", ids))

	if len(ids) == 0 {
		svc.logger.Warn("No IDs provided for role deletion")
		return nil
	}

	err := svc.repo.DeleteByIds(ids)
	if err != nil {
		svc.logger.Error("Failed to delete roles by IDs",
			zap.Int64s("ids", ids),
			zap.Error(err))
		return fmt.Errorf("error deleting roles with ids: %w", err)
	}

	svc.logger.Info("Roles deleted successfully", zap.Int64s("ids", ids))
	return nil
}
