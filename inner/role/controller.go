package role

import (
	"errors"
	"idm/inner/common"
	"idm/inner/web"
	"strconv"
	"strings"

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
	FindById(id int64) (Response, error)
	CreateRole(request CreateRequest) (int64, error)
	FindAll() ([]Response, error)
	FindByIds(ids []int64) ([]Response, error)
	DeleteById(id int64) error
	DeleteByIds(ids []int64) error
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
		return common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())
	}

	// логируем тело запроса
	c.logger.Debug("create role: received request", zap.Any("request", request))

	// вызываем метод CreateRole сервиса role.Service
	newRoleId, err := c.roleService.CreateRole(request)
	if err != nil {
		switch {
		// Handle validation errors
		case errors.As(err, &common.RequestValidationError{}) || errors.As(err, &common.AlreadyExistsError{}):
			c.logger.Warn("Create role validation or conflict error",
				zap.String("name", request.Name),
				zap.Error(err),
				zap.String("ip", ctx.IP()))
			return common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())

		// Handle other errors
		default:
			c.logger.Error("create role internal error",
				zap.String("name", request.Name),
				zap.Error(err),
				zap.String("ip", ctx.IP()))
			return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
		}
	}

	c.logger.Info("Role created successfully",
		zap.String("name", request.Name),
		zap.Int64("id", newRoleId),
		zap.String("ip", ctx.IP()))

	// функция OkResponse() формирует и направляет ответ в случае успеха
	return common.OkResponse(ctx, newRoleId)
}

func (c *Controller) FindRoleById(ctx *fiber.Ctx) error {
	c.logger.Debug("Received find role by ID request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid role ID")
	}

	role, err := c.roleService.FindById(id)
	if err != nil {
		c.logger.Error("Invalid role ID in get request",
			zap.String("id_param", ctx.Params("id")),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
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

	roles, err := c.roleService.FindAll()
	if err != nil {
		c.logger.Error("Failed to find all roles",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
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
		Ids []int64 `json:"ids"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		c.logger.Error("Failed to parse find roles by IDs request body",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid request body")
	}

	c.logger.Debug("Parsed find roles by IDs request",
		zap.Int64s("ids", request.Ids),
		zap.String("ip", ctx.IP()))

	roles, err := c.roleService.FindByIds(request.Ids)
	if err != nil {
		c.logger.Error("Failed to find roles by IDs",
			zap.Int64s("ids", request.Ids),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
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

	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		c.logger.Error("Invalid role ID in delete request",
			zap.String("id_param", ctx.Params("id")),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid role ID")
	}

	err = c.roleService.DeleteById(id)
	if err != nil {
		c.logger.Error("Failed to delete role",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
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

	idsParam := ctx.Query("ids")
	if idsParam == "" {
		c.logger.Error("Missing ids parameter in delete request",
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Missing ids parameter")
	}

	idsStr := strings.Split(idsParam, ",")
	var ids []int64
	for _, idStr := range idsStr {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.logger.Error("Invalid role ID in bulk delete request",
				zap.String("id_param", idStr),
				zap.Error(err),
				zap.String("ip", ctx.IP()))
			return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid role ID")
		}
		ids = append(ids, id)
	}

	c.logger.Debug("Parsed delete role by IDs request",
		zap.Int64s("ids", ids),
		zap.String("ip", ctx.IP()))

	err := c.roleService.DeleteByIds(ids)
	if err != nil {
		c.logger.Error("Failed to delete roles by IDs",
			zap.Int64s("ids", ids),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	c.logger.Info("Roles deleted successfully",
		zap.Int64s("ids", ids),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, "Roles deleted successfully")
}
