package common

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2/log"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Общая конфигурация всего приложения
type Config struct {
	DbDriverName   string `validate:"required"`
	Dsn            string `validate:"required"`
	AppName        string `validate:"required"`
	AppVersion     string `validate:"required"`
	LogLevel       string `validate:"required"`
	LogDevelopMode bool   `validate:"required"`
}

// Получение конфигурации из .env файла или переменных окружения
func GetConfig(envFile string) Config {
	var err = godotenv.Load(envFile)
	// если нет файла, то залогируем это и попробуем получить конфиг из переменных окружения
	if err != nil {
		log.Info("Error loading .env file: %v\n", zap.Error(err))
	}

	var cfg = Config{
		DbDriverName:   os.Getenv("DB_DRIVER_NAME"),
		Dsn:            os.Getenv("DB_DSN"),
		AppName:        os.Getenv("APP_NAME"),
		AppVersion:     os.Getenv("APP_VERSION"),
		LogLevel:       os.Getenv("LOG_LEVEL"),
		LogDevelopMode: os.Getenv("LOG_DEVELOP_MODE") == "true",
	}
	err = validator.New().Struct(cfg)
	if err != nil {
		var validateErrs validator.ValidationErrors
		if errors.As(err, &validateErrs) {
			// если конфиг не прошел валидацию, то логируем событие
			log.Error("config validation error: %v", zap.Error(err))
			// Используем panic для возможности тестирования
			panic(fmt.Sprintf("config validation error: %v", err))
		}
	}
	return cfg
}
