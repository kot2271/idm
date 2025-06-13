package role

import (
	"errors"
	"idm/inner/common"
	"idm/inner/web"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	server      *web.Server
	roleService Svc
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

func NewController(server *web.Server, roleService Svc) *Controller {
	return &Controller{
		server:      server,
		roleService: roleService,
	}
}

// функция для регистрации маршрутов
func (c *Controller) RegisterRoutes() {
	// полный маршрут получится "/api/v1/roles"
	api := c.server.GroupApiV1
	api.Post("/roles", c.CreateRole)
	api.Get("/roles/:id", c.FindRoleById)
	api.Get("/roles", c.FindAllRoles)
	api.Post("/roles/ids", c.FindRoleByIds)
	api.Delete("/roles/:id", c.DeleteRoleById)
	api.Delete("/roles", c.DeleteRoleByIds)
}

// функция-хендлер, которая будет вызываться при POST запросе по маршруту "/api/v1/roles"
func (c *Controller) CreateRole(ctx *fiber.Ctx) error {
	// анмаршалим JSON body запроса в структуру CreateRequest
	var request CreateRequest
	if err := ctx.BodyParser(&request); err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())
	}

	// вызываем метод CreateRole сервиса role.Service
	newRoleId, err := c.roleService.CreateRole(request)
	if err != nil {
		switch {
		// Handle validation errors
		case errors.As(err, &common.RequestValidationError{}) || errors.As(err, &common.AlreadyExistsError{}):
			return common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())

		// Handle other errors
		default:
			return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
		}
	}

	// функция OkResponse() формирует и направляет ответ в случае успеха
	return common.OkResponse(ctx, newRoleId)
}

func (c *Controller) FindRoleById(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid role ID")
	}

	role, err := c.roleService.FindById(id)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	return common.OkResponse(ctx, role)
}

func (c *Controller) FindAllRoles(ctx *fiber.Ctx) error {
	roles, err := c.roleService.FindAll()
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	return common.OkResponse(ctx, roles)
}

func (c *Controller) FindRoleByIds(ctx *fiber.Ctx) error {
	var request struct {
		Ids []int64 `json:"ids"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid request body")
	}

	roles, err := c.roleService.FindByIds(request.Ids)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	return common.OkResponse(ctx, roles)
}

func (c *Controller) DeleteRoleById(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid role ID")
	}

	err = c.roleService.DeleteById(id)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	return common.OkResponse(ctx, "Role deleted successfully")
}

func (c *Controller) DeleteRoleByIds(ctx *fiber.Ctx) error {
	idsParam := ctx.Query("ids")
	if idsParam == "" {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Missing ids parameter")
	}

	idsStr := strings.Split(idsParam, ",")
	var ids []int64
	for _, idStr := range idsStr {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid role ID")
		}
		ids = append(ids, id)
	}

	err := c.roleService.DeleteByIds(ids)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	return common.OkResponse(ctx, "Roles deleted successfully")
}
