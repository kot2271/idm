package common

import (
	"os"

	"github.com/joho/godotenv"
)

// Общая конфигурация всего приложения
type Config struct {
	DbDriverName string `validate:"required"`
	Dsn          string `validate:"required"`
}

// Получение конфигурации из .env файла или переменных окружения
func GetConfig(envFile string) Config {
	_ = godotenv.Load(envFile)
	var cfg = Config{
		DbDriverName: os.Getenv("DB_DRIVER_NAME"),
		Dsn:          os.Getenv("DB_DSN"),
	}
	return cfg
}
