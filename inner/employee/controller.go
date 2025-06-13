package employee

import (
	"errors"
	"idm/inner/common"
	"idm/inner/web"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	server          *web.Server
	employeeService Svc
}

// интерфейс сервиса employee.Service
type Svc interface {
	FindById(id int64) (Response, error)
	CreateEmployee(request CreateRequest) (int64, error)
	DeleteById(id int64) error
	FindAll() ([]Response, error)
	FindByIds(ids []int64) ([]Response, error)
	DeleteByIds(ids []int64) error
}

func NewController(server *web.Server, employeeService Svc) *Controller {
	return &Controller{
		server:          server,
		employeeService: employeeService,
	}
}

// функция для регистрации маршрутов
func (c *Controller) RegisterRoutes() {
	// полный маршрут получится "/api/v1/employees"
	api := c.server.GroupApiV1
	api.Post("/employees", c.CreateEmployee)
	api.Get("/employees/:id", c.GetEmployee)
	api.Delete("/employees/:id", c.DeleteEmployee)
	api.Get("/employees", c.FindAllEmployee)
	api.Post("/employees/ids", c.FindEmployeeByIds)
	api.Delete("/employees", c.DeleteEmployeeByIds)
}

// функция-хендлер, которая будет вызываться при POST запросе по маршруту "/api/v1/employees"
func (c *Controller) CreateEmployee(ctx *fiber.Ctx) error {

	// анмаршалим JSON body запроса в структуру CreateRequest
	var request CreateRequest
	if err := ctx.BodyParser(&request); err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())
	}

	// вызываем метод CreateEmployee сервиса employee.Service
	var newEmployeeId, err = c.employeeService.CreateEmployee(request)
	if err != nil {
		switch {

		// если сервис возвращает ошибку RequestValidationError или AlreadyExistsError,
		// то мы возвращаем ответ с кодом 400 (BadRequest)
		case errors.As(err, &common.RequestValidationError{}) || errors.As(err, &common.AlreadyExistsError{}):
			return common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())

		// если сервис возвращает другую ошибку, то мы возвращаем ответ с кодом 500 (InternalServerError)
		default:
			return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
		}
	}

	// функция OkResponse() формирует и направляет ответ в случае успеха
	return common.OkResponse(ctx, newEmployeeId)
}

func (c *Controller) GetEmployee(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid employee ID")
	}

	employee, err := c.employeeService.FindById(id)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	return common.OkResponse(ctx, employee)
}

func (c *Controller) DeleteEmployee(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid employee ID")
	}

	err = c.employeeService.DeleteById(id)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	return common.OkResponse(ctx, "Employee deleted successfully")
}

func (c *Controller) FindAllEmployee(ctx *fiber.Ctx) error {
	employees, err := c.employeeService.FindAll()
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}
	return common.OkResponse(ctx, employees)
}

func (c *Controller) FindEmployeeByIds(ctx *fiber.Ctx) error {
	var request struct {
		Ids []int64 `json:"ids"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid request body")
	}

	employees, err := c.employeeService.FindByIds(request.Ids)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}
	return common.OkResponse(ctx, employees)
}

func (c *Controller) DeleteEmployeeByIds(ctx *fiber.Ctx) error {
	idsParam := ctx.Query("ids")
	if idsParam == "" {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Missing ids parameter")
	}

	idsStr := strings.Split(idsParam, ",")
	var ids []int64
	for _, idStr := range idsStr {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid employee ID")
		}
		ids = append(ids, id)
	}

	err := c.employeeService.DeleteByIds(ids)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	return common.OkResponse(ctx, "Employees deleted successfully")
}
