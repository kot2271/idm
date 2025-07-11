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
	// полный маршрут получится "/api/v1/admin/employees"
	// Маршруты для чтения (доступны пользователям с ролью IDM_ADMIN или IDM_USER)
	c.server.GroupApiV1User.Get("/employees/page", c.FindEmployeesWithPagination)
	c.server.GroupApiV1User.Get("/employees/:id", c.GetEmployee)
	c.server.GroupApiV1User.Get("/employees", c.FindAllEmployee)
	c.server.GroupApiV1User.Post("/employees/ids", c.FindEmployeeByIds)

	// полный маршрут получится "/api/v1/employees"
	// Маршруты для создания, изменения и удаления (доступны только администраторам)
	c.server.GroupApiV1Admin.Post("/employees", c.CreateEmployee)
	c.server.GroupApiV1Admin.Delete("/employees/:id", c.DeleteEmployee)
	c.server.GroupApiV1Admin.Delete("/employees", c.DeleteEmployeeByIds)

	c.logger.Info("Employee routes registered successfully")
}

// функция-хендлер, которая будет вызываться при POST запросе по маршруту "/api/v1/admin/employees"
// CreateEmployee Creates a new employee
//
// @Security		OAuth2AccessCode[write]
//
//	@Summary		Create an employee
//	@Description	Create a new employee
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			request	body		employee.CreateRequest	true	"create employee request"
//	@Success		200		{object}	common.Response[any]	"Employee successfully created"
//	@Failure		400		{object}	common.Response[any]	"Incorrect data format in request"
//	@Failure		500		{object}	common.Response[any]	"Internal server error"
//	@Router			/employees [post]
func (c *Controller) CreateEmployee(ctx *fiber.Ctx) error {
	c.logger.Info("Received create employee request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	// Получаем роли пользователя для логирования
	userRoles := web.GetUserRoles(ctx)
	c.logger.Debug("User roles", zap.Strings("roles", userRoles))

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

// GetEmployee получает сотрудника по ID
//
// @Security		OAuth2AccessCode[read]
//
//	@Summary		Get employee by ID
//	@Description	Accessing data about an employee using their ID
//	@Tags			employees
//	@Produce		json
//	@Param			id	path		int						true	"Employee ID"
//	@Success		200	{object}	common.Response[any]	"Employee information"
//	@Failure		400	{object}	common.Response[any]	"Invalid employee ID"
//	@Failure		404	{object}	common.Response[any]	"Employee not found
//	@Router			/employees/{id} [get]
func (c *Controller) GetEmployee(ctx *fiber.Ctx) error {
	c.logger.Debug("Received get employee request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	userRoles := web.GetUserRoles(ctx)
	c.logger.Debug("User roles", zap.Strings("roles", userRoles))

	idParam := ctx.Params("id")
	if idParam == "" {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Employee ID is required")
	}

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
		return c.handleFindEmployeeError(ctx, err, id)
	}

	c.logger.Debug("Employee retrieved successfully",
		zap.Int64("id", id),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, employee)
}

// DeleteEmployee удаляет сотрудника по ID
//
// @Security		OAuth2AccessCode[write]
//
//	@Summary		Delete employee
//	@Description	Removing an employee from the system by their ID
//	@Tags			employees
//	@Param			id	path		int						true	"ID сотрудника"
//	@Success		200	{object}	common.Response[any]	"Employee deleted successfully"
//	@Failure		400	{object}	common.Response[any]	"Invalid employee ID"
//	@Failure		404	{object}	common.Response[any]	"Employee doesn't exists"
//	@Failure		500	{object}	common.Response[any]	"Error when deleting an employee"
//	@Router			/employees/{id} [delete]
func (c *Controller) DeleteEmployee(ctx *fiber.Ctx) error {
	c.logger.Info("Received delete employee request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	userRoles := web.GetUserRoles(ctx)
	c.logger.Debug("User roles", zap.Strings("roles", userRoles))

	idParam := ctx.Params("id")
	if idParam == "" {
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Employee ID is required")
	}

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.logger.Error("Invalid employee ID format",
			zap.String("id_param", idParam),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Invalid employee ID format")
	}

	// context.Context нужен для поддержки отмены, дедлайнов и трейсинга запросов к БД.
	err = c.employeeService.DeleteById(ctx.Context(), id)
	if err != nil {
		return c.handleDeleteEmployeeError(ctx, err, id)
	}

	c.logger.Info("Employee deleted successfully",
		zap.Int64("id", id),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, fiber.Map{"message": "Employee deleted successfully"})
}

// FindAllEmployee получает всех сотрудников
//
// @Security		OAuth2AccessCode[read]
//
//	@Summary		Get all employees
//	@Description	Obtain a list of all employees.
//	@Tags			employees
//	@Produce		json
//	@Success		200	{array}		common.Response[any]	"List of employees"
//	@Failure		500	{object}	common.Response[any]	"Error when getting the list of employees"
//	@Router			/employees [get]
func (c *Controller) FindAllEmployee(ctx *fiber.Ctx) error {
	c.logger.Debug("Received find all employees request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	userRoles := web.GetUserRoles(ctx)
	c.logger.Debug("User roles", zap.Strings("roles", userRoles))

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

// FindEmployeeByIds получает сотрудников по списку ID
//
// @Security		OAuth2AccessCode[read]
//
//	@Summary		Get employees by ID list
//	@Description	Obtaining information about employees based on their ID numbers
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			ids	body		[]int64					true	"List of employee IDs"
//	@Success		200	{array}		Response				"List of employees"
//	@Failure		400	{object}	common.Response[any]	"Invalid request body"
//	@Failure		500	{object}	common.Response[any]	"Error searching for employees"
//	@Router			/employees/ids [post]
func (c *Controller) FindEmployeeByIds(ctx *fiber.Ctx) error {
	c.logger.Debug("Received find employees by IDs request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	userRoles := web.GetUserRoles(ctx)
	c.logger.Debug("User roles", zap.Strings("roles", userRoles))

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

// FindEmployeesWithPagination получает сотрудников с пагинацией
//
// @Security		OAuth2AccessCode[read]
//
//	@Summary		Get employees with pagination
//	@Description	Obtaining a list of employees with support for page-by-page output
//	@Tags			employees
//	@Produce		json
//	@Param			pageNumber	query		int						false	"Page number"				default(1)
//	@Param			pageSize	query		int						false	"Number of items on page"	default(10)
//	@Param			textFilter	query		string					false	"Text filter (name, email)"	example("John")
//	@Success		200			{object}	PageResponse			"List of employees with pagination"
//	@Failure		400			{object}	common.Response[any]	"Error when getting paginated employees"
//	@Router			/employees/page [get]
func (c *Controller) FindEmployeesWithPagination(ctx *fiber.Ctx) error {
	c.logger.Debug("Received paginated employees request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("query", ctx.OriginalURL()),
		zap.String("ip", ctx.IP()))

	userRoles := web.GetUserRoles(ctx)
	c.logger.Debug("User roles", zap.Strings("roles", userRoles))

	// Получение параметров из query string
	pageNumberStr := ctx.Query("pageNumber", "1")
	pageSizeStr := ctx.Query("pageSize", "10")
	textFilter := ctx.Query("textFilter", "")

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

	// запрос пагинации с фильтром
	pageRequest := PageRequest{
		PageNumber: pageNumber,
		PageSize:   pageSize,
		TextFilter: textFilter,
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
			zap.String("textFilter", textFilter),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusBadRequest, "Error when getting paginated employees")
	}

	c.logger.Debug("Paginated employees retrieved successfully",
		zap.Int("pageNumber", pageResponse.PageNumber),
		zap.Int("pageSize", pageResponse.PageSize),
		zap.String("textFilter", textFilter),
		zap.Int64("totalCount", pageResponse.TotalCount),
		zap.Int("totalPages", pageResponse.TotalPages),
		zap.Int("dataCount", len(pageResponse.Data)),
		zap.String("ip", ctx.IP()))

	return common.OkResponse(ctx, pageResponse)
}

// DeleteEmployeeByIds удаляет сотрудников по списку ID
//
// @Security		OAuth2AccessCode[write]
//
//	@Summary		Delete employees by ID list
//	@Description	Removing employees from the system by their ID list
//	@Tags			employees
//	@Accept			json
//	@Param			ids	body	[]int64	true	"List of employee IDs to be deleted"
//	@Success		200	"Employees deleted successfully"
//	@Failure		400	{object}	common.Response[any]	"Incorrect data format in the request"
//	@Failure		500	{object}	common.Response[any]	"Error when deleting employees"
//	@Router			/employees [delete]
func (c *Controller) DeleteEmployeeByIds(ctx *fiber.Ctx) error {
	c.logger.Info("Received delete employees by IDs request",
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
		zap.String("ip", ctx.IP()))

	userRoles := web.GetUserRoles(ctx)
	c.logger.Debug("User roles", zap.Strings("roles", userRoles))

	var request struct {
		Ids []int64 `json:"ids" validate:"required,min=1"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		c.logger.Error("Missing ids parameter in delete request",
			zap.Error(err),
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

// обрабатывает ошибки при создании сотрудника
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

// обрабатывает ошибки при поиске сотрудника
func (c *Controller) handleFindEmployeeError(ctx *fiber.Ctx, err error, id int64) error {
	switch {
	case errors.As(err, &common.NotFoundError{}):
		c.logger.Warn("Employee not found",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusNotFound, "Employee not found")

	default:
		c.logger.Error("Find employee internal error",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Internal server error")
	}
}

// обрабатывает ошибки при удалении сотрудника
func (c *Controller) handleDeleteEmployeeError(ctx *fiber.Ctx, err error, id int64) error {
	switch {
	case errors.As(err, &common.NotFoundError{}):
		c.logger.Warn("Employee not found for deletion",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusNotFound, "Employee not found")

	default:
		c.logger.Error("Delete employee internal error",
			zap.Int64("id", id),
			zap.Error(err),
			zap.String("ip", ctx.IP()))
		return common.ErrResponse(ctx, fiber.StatusInternalServerError, "Internal server error")
	}
}
