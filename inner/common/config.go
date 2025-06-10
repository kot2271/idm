package common

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

// Общая конфигурация всего приложения
type Config struct {
	DbDriverName string `validate:"required"`
	Dsn          string `validate:"required"`
	AppName      string `validate:"required"`
	AppVersion   string `validate:"required"`
}

// Получение конфигурации из .env файла или переменных окружения
func GetConfig(envFile string) Config {
	var err = godotenv.Load(envFile)
	// если нет файла, то залогируем это и попробуем получить конфиг из переменных окружения
	if err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
	}
	var cfg = Config{
		DbDriverName: os.Getenv("DB_DRIVER_NAME"),
		Dsn:          os.Getenv("DB_DSN"),
		AppName:      os.Getenv("APP_NAME"),
		AppVersion:   os.Getenv("APP_VERSION"),
	}
	err = validator.New().Struct(cfg)
	if err != nil {
		var validateErrs validator.ValidationErrors
		if errors.As(err, &validateErrs) {
			// если конфиг не прошел валидацию, то паникуем
			panic(fmt.Sprintf("config validation error: %v", err))
		}
	}
	return cfg
}
