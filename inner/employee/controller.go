package employee

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
	server          *web.Server
	employeeService Svc
	logger          *common.Logger
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

func NewController(server *web.Server, employeeService Svc, logger *common.Logger) *Controller {
	return &Controller{
		server:          server,
		employeeService: employeeService,
		logger:          logger,
	}
}

// функция для регистрации маршрутов
func (c *Controller) RegisterRoutes() {
	c.logger.Info("Registering employee routes")
	// полный маршрут получится "/api/v1/employees"
	api := c.server.GroupApiV1
	api.Post("/employees", c.CreateEmployee)
	api.Get("/employees/:id", c.GetEmployee)
	api.Delete("/employees/:id", c.DeleteEmployee)
	api.Get("/employees", c.FindAllEmployee)
	api.Post("/employees/ids", c.FindEmployeeByIds)
	api.Delete("/employees", c.DeleteEmployeeByIds)
	c.logger.Info("Employee routes registered successfully")
}

// функция-хендлер, которая будет вызываться при POST запросе по маршруту "/api/v1/employees"
func (c *Controller) CreateEmployee(ctx *fiber.Ctx) error {
	c.logger.Info("Received create employee request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	// анмаршалим JSON body запроса в структуру CreateRequest
	var request CreateRequest
	if err := ctx.BodyParser(&request); err != nil {
		c.logger.Error("Failed to parse create employee request body",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())
	}

	// логируем тело запроса
	c.logger.Debug("create employee: received request", zap.Any("request", request))

	// вызываем метод CreateEmployee сервиса employee.Service
	var newEmployeeId, err = c.employeeService.CreateEmployee(request)
	if err != nil {
		switch {

		// если сервис возвращает ошибку RequestValidationError или AlreadyExistsError,
		// то мы возвращаем ответ с кодом 400 (BadRequest)
		case errors.As(err, &common.RequestValidationError{}) || errors.As(err, &common.AlreadyExistsError{}):
			c.logger.Warn("Create employee validation or conflict error",
				zap.String("name", request.Name),
				zap.Error(err),
				zap.String("ip", ctx.IP()))
			return common.ErrResponse(ctx, fiber.StatusBadRequest, err.Error())

		// если сервис возвращает другую ошибку, то мы возвращаем ответ с кодом 500 (InternalServerError)
		default:
			c.logger.Error("create employee internal error",
				zap.String("name", request.Name),
				zap.Error(err),
				zap.String("ip", ctx.IP()))
			return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
		}
	}

	c.logger.Info("Employee created successfully",
		zap.String("name", request.Name),
		zap.Int64("id", newEmployeeId),
		zap.String("ip", ctx.IP()))

	// функция OkResponse() формирует и направляет ответ в случае успеха
	return common.OkResponse(ctx, newEmployeeId)
}

func (c *Controller) GetEmployee(ctx *fiber.Ctx) error {
	c.logger.Debug("Received get employee request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid employee ID")
	}

	employee, err := c.employeeService.FindById(id)
	if err != nil {
		c.logger.Error("Invalid employee ID in get request",
			zap.String("id_param", ctx.Params("id")),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	c.logger.Debug("Employee retrieved successfully",
		zap.Int64("id", id),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, employee)
}

func (c *Controller) DeleteEmployee(ctx *fiber.Ctx) error {
	c.logger.Info("Received delete employee request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		c.logger.Error("Invalid employee ID in delete request",
			zap.String("id_param", ctx.Params("id")),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid employee ID")
	}

	err = c.employeeService.DeleteById(id)
	if err != nil {
		c.logger.Error("Failed to delete employee",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	c.logger.Info("Employee deleted successfully",
		zap.Int64("id", id),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, "Employee deleted successfully")
}

func (c *Controller) FindAllEmployee(ctx *fiber.Ctx) error {
	c.logger.Debug("Received find all employees request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	employees, err := c.employeeService.FindAll()
	if err != nil {
		c.logger.Error("Failed to find all employees",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	c.logger.Debug("All employees retrieved successfully",
		zap.Int("count", len(employees)),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, employees)
}

func (c *Controller) FindEmployeeByIds(ctx *fiber.Ctx) error {
	c.logger.Debug("Received find employees by IDs request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	var request struct {
		Ids []int64 `json:"ids"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		c.logger.Error("Failed to parse find employees by IDs request body",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid request body")
	}

	c.logger.Debug("Parsed find employees by IDs request",
		zap.Int64s("ids", request.Ids),
		zap.String("ip", ctx.IP()))

	employees, err := c.employeeService.FindByIds(request.Ids)
	if err != nil {
		c.logger.Error("Failed to find employees by IDs",
			zap.Int64s("ids", request.Ids),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	c.logger.Debug("Employees found by IDs successfully",
		zap.Int64s("ids", request.Ids),
		zap.Int("found_count", len(employees)),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, employees)
}

func (c *Controller) DeleteEmployeeByIds(ctx *fiber.Ctx) error {
	c.logger.Info("Received delete employees by IDs request",
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
			c.logger.Error("Invalid employee ID in bulk delete request",
				zap.String("id_param", idStr),
				zap.Error(err),
				zap.String("ip", ctx.IP()))
			return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid employee ID")
		}
		ids = append(ids, id)
	}

	c.logger.Debug("Parsed delete employees by IDs request",
		zap.Int64s("ids", ids),
		zap.String("ip", ctx.IP()))

	err := c.employeeService.DeleteByIds(ids)
	if err != nil {
		c.logger.Error("Failed to delete employees by IDs",
			zap.Int64s("ids", ids),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, err.Error())
	}

	c.logger.Info("Employees deleted successfully",
		zap.Int64s("ids", ids),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, "Employees deleted successfully")
}
