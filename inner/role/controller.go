package role

import (
	"context"
	"errors"
	"idm/inner/common"
	"idm/inner/web"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Controller struct {
	server      *web.Server
	roleService Svc
	logger      *common.Logger
}

// интерфейс сервиса role.Service
type Svc interface {
	FindById(ctx context.Context, id int64) (Response, error)
	CreateRole(ctx context.Context, request CreateRequest) (int64, error)
	FindAll(ctx context.Context) ([]Response, error)
	FindByIds(ctx context.Context, ids []int64) ([]Response, error)
	DeleteById(ctx context.Context, id int64) error
	DeleteByIds(ctx context.Context, ids []int64) error
}

func NewController(server *web.Server, roleService Svc, logger *common.Logger) *Controller {
	return &Controller{
		server:      server,
		roleService: roleService,
		logger:      logger,
	}
}

// функция для регистрации маршрутов
func (c *Controller) RegisterRoutes() {
	c.logger.Info("Registering role routes")
	// полный маршрут получится "/api/v1/roles"
	api := c.server.GroupApiV1
	api.Post("/roles", c.CreateRole)
	api.Get("/roles/:id", c.FindRoleById)
	api.Get("/roles", c.FindAllRoles)
	api.Post("/roles/ids", c.FindRoleByIds)
	api.Delete("/roles/:id", c.DeleteRoleById)
	api.Delete("/roles", c.DeleteRoleByIds)
	c.logger.Info("Role routes registered successfully")
}

// функция-хендлер, которая будет вызываться при POST запросе по маршруту "/api/v1/roles"
func (c *Controller) CreateRole(ctx *fiber.Ctx) error {
	c.logger.Info("Received create role request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	// анмаршалим JSON body запроса в структуру CreateRequest
	var request CreateRequest
	if err := ctx.BodyParser(&request); err != nil {
		c.logger.Error("Failed to parse create role request body",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Incorrect data format in request")
	}

	// логируем тело запроса
	c.logger.Debug("create role: received request", zap.Any("request", request))

	// вызываем метод CreateRole сервиса role.Service
	newRoleId, err := c.roleService.CreateRole(ctx.Context(), request)
	if err != nil {
		return c.handleCreateRoleError(ctx, err, request)
	}

	c.logger.Info("Role created successfully",
		zap.String("name", request.Name),
		zap.Int64("id", newRoleId),
		zap.String("ip", ctx.IP()))

	// функция OkResponse() формирует и направляет ответ в случае успеха
	return common.OkResponse(ctx, fiber.Map{
		"id":      newRoleId,
		"message": "Role successfully created",
	})
}

// обрабатывает ошибки при создании роли
func (c *Controller) handleCreateRoleError(ctx *fiber.Ctx, err error, request CreateRequest) error {
	switch {
	case errors.As(err, &common.RequestValidationError{}):
		c.logger.Warn("Create role validation error",
			zap.String("name", request.Name),
			zap.Error(err),
			zap.String("ip", ctx.IP()))

		var validationErr common.RequestValidationError
		errors.As(err, &validationErr)

		if validationErr.Data != nil {
			return common.ErrResponse(ctx, fiber.StatusBadRequest, "Data validation error", validationErr.Data)
		}
		return common.ErrResponse(ctx, fiber.StatusBadRequest, validationErr.Message)

	case errors.As(err, &common.AlreadyExistsError{}):
		c.logger.Warn("Create role conflict error",
			zap.String("name", request.Name),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusConflict, err.Error())

	default:
		c.logger.Error("Create role internal error",
			zap.String("name", request.Name),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Internal server error")
	}
}

func (c *Controller) FindRoleById(ctx *fiber.Ctx) error {
	c.logger.Debug("Received find role by ID request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	idParam := ctx.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.logger.Error("Invalid role ID format",
			zap.String("id", idParam),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid role ID format")
	}

	role, err := c.roleService.FindById(ctx.Context(), id)
	if err != nil {
		c.logger.Error("Failed to find role by ID",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusNotFound, "Role not found")
	}

	c.logger.Debug("Role retrieved successfully",
		zap.Int64("id", id),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, role)
}

func (c *Controller) FindAllRoles(ctx *fiber.Ctx) error {
	c.logger.Debug("Received find all roles request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	roles, err := c.roleService.FindAll(ctx.Context())
	if err != nil {
		c.logger.Error("Failed to find all roles",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Error when getting the list of roles")
	}

	c.logger.Debug("All roles retrieved successfully",
		zap.Int("count", len(roles)),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, roles)
}

func (c *Controller) FindRoleByIds(ctx *fiber.Ctx) error {
	c.logger.Debug("Received find roles by IDs request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	var request struct {
		Ids []int64 `json:"ids" validate:"required,min=1"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		c.logger.Error("Failed to parse find roles by IDs request body",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Incorrect data format in request")
	}

	if len(request.Ids) == 0 {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "The ID list cannot be empty.")
	}

	c.logger.Debug("Parsed find roles by IDs request",
		zap.Int64s("ids", request.Ids),
		zap.String("ip", ctx.IP()))

	roles, err := c.roleService.FindByIds(ctx.Context(), request.Ids)
	if err != nil {
		c.logger.Error("Failed to find roles by IDs",
			zap.Int64s("ids", request.Ids),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Error when searching for roles")
	}

	c.logger.Debug("Roles found by IDs successfully",
		zap.Int64s("ids", request.Ids),
		zap.Int("found_count", len(roles)),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, roles)
}

func (c *Controller) DeleteRoleById(ctx *fiber.Ctx) error {
	c.logger.Info("Received delete role by ID request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	idParam := ctx.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.logger.Error("Invalid role ID in delete request",
			zap.String("id", idParam),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid role ID format")
	}

	c.logger.Info("Received delete role request",
		zap.Int64("id", id),
		zap.String("ip", ctx.IP()))

	err = c.roleService.DeleteById(ctx.Context(), id)
	if err != nil {
		c.logger.Error("Failed to delete role",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Error when deleting a role")
	}

	c.logger.Info("Role deleted successfully",
		zap.Int64("id", id),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, "Role deleted successfully")
}

func (c *Controller) DeleteRoleByIds(ctx *fiber.Ctx) error {
	c.logger.Info("Received delete role by IDs request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	var request struct {
		Ids []int64 `json:"ids" validate:"required,min=1"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		c.logger.Error("Failed to parse delete role by IDs request body",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Incorrect data format in request")
	}

	if len(request.Ids) == 0 {
		c.logger.Warn("The ID list cannot be empty.",
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "The ID list cannot be empty.")
	}

	c.logger.Debug("Parsed delete role by IDs request",
		zap.Int64s("ids", request.Ids),
		zap.String("ip", ctx.IP()))

	err := c.roleService.DeleteByIds(ctx.Context(), request.Ids)
	if err != nil {
		c.logger.Error("Failed to delete roles by IDs",
			zap.Int64s("ids", request.Ids),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Error when deleting roles")
	}

	c.logger.Info("Roles deleted successfully",
		zap.Int64s("ids", request.Ids),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, fiber.Map{"message": "Roles deleted successfully"})
}
