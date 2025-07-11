package common

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

type Response[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"error"`
	Data    T      `json:"data,omitempty"`
} // @name Response

func ErrResponse(
	c *fiber.Ctx,
	code int,
	message string,
	data ...any,
) error {
	response := Response[any]{
		Success: false,
		Message: message,
	}
	if len(data) > 0 {
		response.Data = data[0]
	}
	return c.Status(code).JSON(response)
}

func OkResponse[T any](
	c *fiber.Ctx,
	data T,
) error {
	return c.JSON(&Response[T]{
		Success: true,
		Data:    data,
	})
}

// ValidationErrorResponse формирует ответ с ошибками валидации
func ValidationErrorResponse(ctx *fiber.Ctx, validationErr error) error {
	// Попытаемся распарсить ошибку валидации как JSON
	var validationDetails any
	if jsonErr := json.Unmarshal([]byte(validationErr.Error()), &validationDetails); jsonErr == nil {
		// Если это JSON, используем его как детали
		return ErrResponse(ctx, fiber.StatusBadRequest, "Data validation error", validationDetails)
	}

	// Если не JSON, просто возвращаем текст ошибки
	return ErrResponse(ctx, fiber.StatusBadRequest, validationErr.Error())
}

// NewNotFoundError создаёт новую ошибку "not found"
func NewNotFoundError(message string) error {
	return NotFoundError{Message: message}
}
