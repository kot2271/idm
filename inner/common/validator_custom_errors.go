package common

type RequestValidationError struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (err RequestValidationError) Error() string {
	return err.Message
}

type AlreadyExistsError struct {
	Message string `json:"message"`
}

func (err AlreadyExistsError) Error() string {
	return err.Message
}

// NotFoundError представляет ошибку, когда сущность не найдена
type NotFoundError struct {
	Message string `json:"message"`
}

func (err NotFoundError) Error() string {
	return err.Message
}
