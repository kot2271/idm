package web

import "github.com/gofiber/fiber"

// структуа веб-сервера
type Server struct {
	App        *fiber.App
	GroupApiV1 fiber.Router
}

// функция-конструктор
func NewServer() *Server {

	// создаём новый веб-вервер
	app := fiber.New()

	// создаём группу "/api"
	groupApi := app.Group("/api")

	// создаём подгруппу "api/v1"
	groupApiV1 := groupApi.Group("/v1")
	return &Server{
		App:        app,
		GroupApiV1: groupApiV1,
	}
}
