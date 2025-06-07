package role

import (
	"errors"
	"idm/inner/common"
	"idm/inner/web"

	"github.com/gofiber/fiber"
)

type Controller struct {
	server      *web.Server
	roleService Svc
}

// интерфейс сервиса role.Service
type Svc interface {
	FindById(id int64) (Response, error)
	CreateRole(request CreateRequest) (int64, error)
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
	c.server.GroupApiV1.Post("/roles", c.CreateRole)
}

// функция-хендлер, которая будет вызываться при POST запросе по маршруту "/api/v1/roles"
func (c *Controller) CreateRole(ctx *fiber.Ctx) {
	// анмаршалим JSON body запроса в структуру CreateRequest
	var request CreateRequest
	if err := ctx.BodyParser(&request); err != nil {
		_ = common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())
		return
	}

	// вызываем метод CreateRole сервиса role.Service
	newRoleId, err := c.roleService.CreateRole(request)
	if err != nil {
		switch {
		// Handle validation errors
		case errors.As(err, &common.RequestValidationError{}) || errors.As(err, &common.AlreadyExistsError{}):
			_ = common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())

		// Handle other errors
		default:
			_ = common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
		}
		return
	}

	// функция OkResponse() формирует и направляет ответ в случае успеха
	if err := common.OkResponse(ctx, newRoleId); err != nil {
		_ = common.ErrResponse(ctx, fiber.StatusInternalServerError, "error returning created role id")
		return
	}
}
