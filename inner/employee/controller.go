package employee

import (
	"context"
	"errors"
	"idm/inner/common"
	"idm/inner/web"
	"strconv"
	"time"

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
	FindById(ctx context.Context, id int64) (Response, error)
	CreateEmployee(ctx context.Context, request CreateRequest) (int64, error)
	DeleteById(ctx context.Context, id int64) error
	FindAll(ctx context.Context) ([]Response, error)
	FindByIds(ctx context.Context, ids []int64) ([]Response, error)
	DeleteByIds(ctx context.Context, ids []int64) error
	FindWithPagination(ctx context.Context, request PageRequest) (PageResponse, error)
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
	api.Get("/employees/page", c.FindEmployeesWithPagination)
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
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Incorrect data format in request")
	}

	// логируем тело запроса
	c.logger.Debug("create employee: received request", zap.Any("request", request))

	// context.Context нужен для поддержки отмены, дедлайнов и трейсинга запросов к БД.
	newEmployeeId, err := c.employeeService.CreateEmployee(ctx.Context(), request)
	if err != nil {
		return c.handleCreateEmployeeError(ctx, err, request)
	}

	c.logger.Info("Employee created successfully",
		zap.String("name", request.Name),
		zap.Int64("id", newEmployeeId),
		zap.String("ip", ctx.IP()))

	// функция OkResponse() формирует и направляет ответ в случае успеха
	return common.OkResponse(ctx, fiber.Map{
		"id":      newEmployeeId,
		"message": "Employee successfully created",
	})
}

// handleCreateEmployeeError обрабатывает ошибки при создании сотрудника
func (c *Controller) handleCreateEmployeeError(ctx *fiber.Ctx, err error, request CreateRequest) error {
	switch {
	// Обработка ошибок валидации
	case errors.As(err, &common.RequestValidationError{}):
		c.logger.Warn("Create employee validation error",
			zap.String("name", request.Name),
			zap.Error(err),
			zap.String("ip", ctx.IP()))

		// Получаем детали ошибки валидации
		var validationErr common.RequestValidationError
		errors.As(err, &validationErr)

		if validationErr.Data != nil {
			return common.ErrResponse(ctx, fiber.StatusBadRequest, "Data validation error", validationErr.Data)
		}
		return common.ErrResponse(ctx, fiber.StatusBadRequest, validationErr.Message)

	// Обработка ошибок существования
	case errors.As(err, &common.AlreadyExistsError{}):
		c.logger.Warn("Create employee conflict error",
			zap.String("name", request.Name),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusConflict, err.Error())

	// Обработка других ошибок
	default:
		c.logger.Error("Create employee internal error",
			zap.String("name", request.Name),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Internal server error")
	}
}

func (c *Controller) GetEmployee(ctx *fiber.Ctx) error {
	c.logger.Debug("Received get employee request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	idParam := ctx.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.logger.Error("Invalid employee ID format",
			zap.String("id", idParam),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid employee ID")
	}

	// context.Context нужен для поддержки отмены, дедлайнов и трейсинга запросов к БД.
	employee, err := c.employeeService.FindById(ctx.Context(), id)
	if err != nil {
		c.logger.Error("Failed to find employee",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusNotFound, "Employee not found")
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

	idParam := ctx.Params("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.logger.Error("Invalid employee ID in delete request",
			zap.String("id_param", ctx.Params("id")),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid employee ID")
	}

	// context.Context нужен для поддержки отмены, дедлайнов и трейсинга запросов к БД.
	err = c.employeeService.DeleteById(ctx.Context(), id)
	if err != nil {
		c.logger.Error("Failed to delete employee",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Error when deleting an employee")
	}

	c.logger.Info("Employee deleted successfully",
		zap.Int64("id", id),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, fiber.Map{"message": "Employee deleted successfully"})
}

func (c *Controller) FindAllEmployee(ctx *fiber.Ctx) error {
	c.logger.Debug("Received find all employees request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	// context.Context нужен для поддержки отмены, дедлайнов и трейсинга запросов к БД.
	employees, err := c.employeeService.FindAll(ctx.Context())
	if err != nil {
		c.logger.Error("Failed to find all employees",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Error when getting the list of employees")
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
		Ids []int64 `json:"ids" validate:"required,min=1"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		c.logger.Error("Failed to parse find employees by IDs request body",
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid request body")
	}

	if len(request.Ids) == 0 {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "The ID list cannot be empty.")
	}

	c.logger.Debug("Parsed find employees by IDs request",
		zap.Int64s("ids", request.Ids),
		zap.String("ip", ctx.IP()))

	// context.Context нужен для поддержки отмены, дедлайнов и трейсинга запросов к БД.
	employees, err := c.employeeService.FindByIds(ctx.Context(), request.Ids)
	if err != nil {
		c.logger.Error("Failed to find employees by IDs",
			zap.Int64s("ids", request.Ids),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Error searching for employees")
	}

	c.logger.Debug("Employees found by IDs successfully",
		zap.Int64s("ids", request.Ids),
		zap.Int("found_count", len(employees)),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, employees)
}

func (c *Controller) FindEmployeesWithPagination(ctx *fiber.Ctx) error {
	c.logger.Debug("Received paginated employees request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("query", ctx.OriginalURL()),
		zap.String("ip", ctx.IP()))

	// Получение параметров из query string
	pageNumberStr := ctx.Query("pageNumber", "1")
	pageSizeStr := ctx.Query("pageSize", "10")

	// Конвертация в числа
	pageNumber, err := strconv.Atoi(pageNumberStr)
	if err != nil {
		c.logger.Error("Invalid pageNumber parameter",
			zap.String("pageNumber", pageNumberStr),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid pageNumber parameter")
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		c.logger.Error("Invalid pageSize parameter",
			zap.String("pageSize", pageSizeStr),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid pageSize parameter")
	}

	// запрос пагинации
	pageRequest := PageRequest{
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}

	// ВАЖНО: Создание нового контекса для работы с БД
	// Это предотвращает проблему с отмененным контекстом в тестах
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pageResponse, err := c.employeeService.FindWithPagination(dbCtx, pageRequest)
	if err != nil {
		c.logger.Error("Failed to find employees with pagination",
			zap.Error(err),
			zap.Int("pageNumber", pageNumber),
			zap.Int("pageSize", pageSize),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Error when getting paginated employees")
	}

	c.logger.Debug("Paginated employees retrieved successfully",
		zap.Int("pageNumber", pageResponse.PageNumber),
		zap.Int("pageSize", pageResponse.PageSize),
		zap.Int64("totalCount", pageResponse.TotalCount),
		zap.Int("totalPages", pageResponse.TotalPages),
		zap.Int("dataCount", len(pageResponse.Data)),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, pageResponse)
}

func (c *Controller) DeleteEmployeeByIds(ctx *fiber.Ctx) error {
	c.logger.Info("Received delete employees by IDs request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	var request struct {
		Ids []int64 `json:"ids" validate:"required,min=1"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		c.logger.Error("Missing ids parameter in delete request",
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Incorrect data format in the request")
	}

	if len(request.Ids) == 0 {
		c.logger.Warn("The ID list cannot be empty.",
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "The ID list cannot be empty.")
	}

	c.logger.Debug("Parsed delete employees by IDs request",
		zap.Int64s("ids", request.Ids),
		zap.String("ip", ctx.IP()))

	// context.Context нужен для поддержки отмены, дедлайнов и трейсинга запросов к БД.
	err := c.employeeService.DeleteByIds(ctx.Context(), request.Ids)
	if err != nil {
		c.logger.Error("Failed to delete employees by IDs",
			zap.Int64s("ids", request.Ids),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Error when deleting employees")
	}

	c.logger.Info("Employees deleted successfully",
		zap.Int64s("ids", request.Ids),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, fiber.Map{"message": "Employees deleted successfully"})
}
