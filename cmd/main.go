package main

import (
	"fmt"
	"idm/inner/common"
	"idm/inner/database"
	"idm/inner/employee"
	"idm/inner/info"
	"idm/inner/role"
	"idm/inner/validator"
	"idm/inner/web"

	"github.com/jmoiron/sqlx"
)

func main() {
	// создаём подключение к базе данных
	database, err := database.ConnectDb()
	// закрываем соединение с базой данных после выхода из функции main
	defer func() {
		if err != nil {
			fmt.Printf("error closing db: %v", err)
		}
	}()
	var server = build(database)
	err = server.App.Listen(":8080")
	if err != nil {
		panic(fmt.Sprintf("http server error: %s", err))
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
