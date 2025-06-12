package main

import (
	"context"
	"idm/inner/common"
	"idm/inner/database"
	"idm/inner/employee"
	"idm/inner/info"
	"idm/inner/role"
	"idm/inner/validator"
	"idm/inner/web"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
)

func main() {
	// подключение к базе данных
	db, err := database.ConnectDb()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	server := build(db)

	// канал для получения системных сигналов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	// запуск сервера в отдельной горутине
	go func() {
		log.Println("Starting server on :8080")
		if err := server.App.Listen(":8080"); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// ожидаем сигнал для завершения работы
	<-quit
	log.Println("Shutting down server...")

	// выполняем graceful shutdown
	gracefulShutdown(server, db)
}

// gracefulShutdown выполняет корректное завершение работы сервера
func gracefulShutdown(server *web.Server, db *sqlx.DB) {
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
				log.Printf("Error closing database connection: %v", err)
			} else {
				log.Println("Database connection closed successfully")
			}
			done <- true
		}()

		// завершаем работу HTTP сервера
		if err := server.App.Shutdown(); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		} else {
			log.Println("Server shutdown completed successfully")
		}
	}()

	// ожидается завершение shutdown или таймаута
	select {
	case <-done:
		log.Println("Graceful shutdown completed")
	case <-ctx.Done():
		log.Println("Shutdown timeout exceeded, forcing exit")
	}
}

// buil функция, конструирующая наш веб-сервер
func build(database *sqlx.DB) *web.Server {
	// читаем конфиги
	var cfg = common.GetConfig(".env")
	// создаём веб-сервер
	var server = web.NewServer()

	// создаём валидатор
	var vld = validator.New()

	// -------------------------
	// Модуль role
	// -------------------------

	// Создаём репозиторий для ролей
	var roleRepo = role.NewRoleRepository(database)

	// Создаём сервис для ролей
	var roleService = role.NewService(roleRepo, vld)

	// Создаём контроллер для ролей
	var roleController = role.NewController(server, roleService)
	roleController.RegisterRoutes()

	// -------------------------
	// Модуль employee
	// -------------------------

	// создаём репозиторий сотрудников
	var employeeRepo = employee.NewEmployeeRepository(database)

	// создаём сервис для сотрудников
	var employeeService = employee.NewService(employeeRepo, vld)

	// создаём контроллер для сотрудников
	var employeeController = employee.NewController(server, employeeService)
	employeeController.RegisterRoutes()

	// -------------------------
	// Модуль info
	// -------------------------

	// контроллер для info
	var infoController = info.NewController(server, cfg, database)
	infoController.RegisterRoutes()

	return server
}
