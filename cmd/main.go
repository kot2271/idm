package main

import (
	"context"
	"idm/docs"
	"idm/inner/common"
	"idm/inner/database"
	"idm/inner/employee"
	"idm/inner/info"
	"idm/inner/role"
	"idm/inner/validator"
	"idm/inner/web"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

//	@title						IDM API documentation
//	@description				Identity Management System API
//	@host						localhost:8080
//	@BasePath					/api/v1
//	@schemes					http https
//
//	@securityDefinitions.basic	BasicAuth
func main() {
	// читаем конфиги
	var cfg = common.GetConfig(".env")

	// Переопределяем версию приложения, которая будет отображаться в swagger UI.
	// Пакет docs и структура SwaggerInfo в нём появятся поле генерации документации
	docs.SwaggerInfo.Version = cfg.AppVersion

	// Создаем логгер
	var logger = common.NewLogger(cfg)

	// Отложенный вызов записи сообщений из буфера в лог. Необходимо вызывать перед выходом из приложения
	defer func() { _ = logger.Sync() }()

	// подключение к базе данных
	db, err := database.ConnectDb()
	if err != nil {
		logger.Fatal("failed to connect to database: %v", zap.Error(err))
	}

	server := build(db, cfg, logger)

	// канал для получения системных сигналов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	// запуск сервера в отдельной горутине
	go func() {
		logger.Info("Starting server on :8080")
		if err := server.App.Listen(":8080"); err != nil {
			logger.Panic("http server error: %s", zap.Error(err))
		}
	}()

	// ожидаем сигнал для завершения работы
	<-quit
	logger.Info("Shutting down server...")

	// выполняем graceful shutdown
	gracefulShutdown(server, db, logger)
}

// gracefulShutdown выполняет корректное завершение работы сервера
func gracefulShutdown(server *web.Server, db *sqlx.DB, logger *common.Logger) {
	// контекст с таймаутом для shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// канал для отслеживания завершения shutdown
	done := make(chan bool, 1)

	// shutdown выполняется в отдельной горутине
	go func() {
		defer func() {
			// закрываем соединение с базой данных
			if err := db.Close(); err != nil {
				logger.Error("Error closing database connection: %v", zap.Error(err))
			} else {
				logger.Info("Database connection closed successfully")
			}
			done <- true
		}()

		// завершаем работу HTTP сервера
		if err := server.App.Shutdown(); err != nil {
			logger.Error("Error during server shutdown: %v", zap.Error(err))
		} else {
			logger.Info("Server shutdown completed successfully")
		}
	}()

	// ожидается завершение shutdown или таймаута
	select {
	case <-done:
		logger.Info("Graceful shutdown completed")
	case <-ctx.Done():
		logger.Info("Shutdown timeout exceeded, forcing exit")
	}
}

// buil функция, конструирующая наш веб-сервер
func build(database *sqlx.DB, cfg common.Config, logger *common.Logger) *web.Server {
	// создаём веб-сервер
	var server = web.NewServer()

	// Добавляем кастомный middleware для логирования
	server.App.Use(web.CustomMiddleware(logger.Logger))

	// создаём валидатор
	var vld = validator.New()

	// -------------------------
	// Модуль role
	// -------------------------

	// Создаём репозиторий для ролей
	var roleRepo = role.NewRoleRepository(database)

	// Создаём сервис для ролей
	var roleService = role.NewService(roleRepo, vld, logger)

	// Создаём контроллер для ролей
	var roleController = role.NewController(server, roleService, logger)
	roleController.RegisterRoutes()

	// -------------------------
	// Модуль employee
	// -------------------------

	// создаём репозиторий сотрудников
	var employeeRepo = employee.NewEmployeeRepository(database)

	// создаём сервис для сотрудников
	var employeeService = employee.NewService(employeeRepo, vld, logger)

	// создаём контроллер для сотрудников
	var employeeController = employee.NewController(server, employeeService, logger)
	employeeController.RegisterRoutes()

	// -------------------------
	// Модуль info
	// -------------------------

	// контроллер для info
	var infoController = info.NewController(server, cfg, database, logger)
	infoController.RegisterRoutes()

	return server
}
