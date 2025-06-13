package web

import "github.com/gofiber/fiber/v2"

// структуа веб-сервера
type Server struct {
	App *fiber.App
	// группа публичного API
	GroupApiV1 fiber.Router
	// группа непубличного API
	GroupInternal fiber.Router
}

// функция-конструктор
func NewServer() *Server {

	// создаём новый веб-вервер
	app := fiber.New()

	groupInternal := app.Group("/internal")

	// создаём группу "/api"
	groupApi := app.Group("/api")

	// создаём подгруппу "api/v1"
	groupApiV1 := groupApi.Group("/v1")
	return &Server{
		App:           app,
		GroupApiV1:    groupApiV1,
		GroupInternal: groupInternal,
	}
}
