package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"idm/inner/common"
	"idm/inner/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetConfig_NoEnvFile(t *testing.T) {
	// Удаляем переменные окружения для чистоты теста
	err := os.Unsetenv("DB_DRIVER_NAME")
	require.NoError(t, err)
	err = os.Unsetenv("DB_DSN")
	require.NoError(t, err)

	// Путь к существующему .env файлу в корне проекта
	// envFilePath := filepath.Join("..", ".env")
	envFilePath := filepath.Join("..", ".env_not_exists")

	// Получаем конфиг без существующего .env файла
	cfg := common.GetConfig(envFilePath)

	assert.Empty(t, cfg.DbDriverName, "Поле DB_DRIVER_NAME должно быть пустым")
	assert.Empty(t, cfg.Dsn, "Поле DB_DSN должно быть пустым")
}

func Test_GetConfig_NoVarsInEnvAndDotEnv(t *testing.T) {
	// Убедимся, что переменные окружения не заданы
	err := os.Unsetenv("DB_DRIVER_NAME")
	require.NoError(t, err)
	err = os.Unsetenv("DB_DSN")
	require.NoError(t, err)

	// Создаем временную директорию
	tempDir := t.TempDir()

	// Путь к фейковому .env файлу
	envFilePath := filepath.Join(tempDir, ".env")

	// Создаем пустой .env файл (или без нужных переменных)
	err = os.WriteFile(envFilePath, []byte(""), 0644)
	assert.NoError(t, err, "Не удалось создать временный .env файл")

	cfg := common.GetConfig(envFilePath)

	assert.Empty(t, cfg.DbDriverName, "Поле DB_DRIVER_NAME должно быть пустым")
	assert.Empty(t, cfg.Dsn, "Поле DB_DSN должно быть пустым")
}

func Test_GetConfig_EnvVarsPresent_ButNotInDotEnv(t *testing.T) {
	// Создаем временную директорию
	tempDir := t.TempDir()

	// Путь к тестовому .env файлу
	envFilePath := filepath.Join(tempDir, ".env")

	// Создаем пустой .env файл (без нужных переменных)
	err := os.WriteFile(envFilePath, []byte(""), 0644)
	assert.NoError(t, err, "Не удалось создать временный .env файл")

	// Устанавливаем переменные окружения
	err = os.Setenv("DB_DRIVER_NAME", "postgres")
	require.NoError(t, err)
	err = os.Setenv("DB_DSN", "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable")
	require.NoError(t, err)

	// Вызываем GetConfig с путём к тестовому .env
	cfg := common.GetConfig(envFilePath)

	// Проверяем, что значения взяты из переменных окружения
	assert.Equal(t, "postgres", cfg.DbDriverName)
	assert.Equal(t, "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable", cfg.Dsn)
}

func Test_ConfigPrioritizesEnv_OverDotEnv(t *testing.T) {
	// Создаем временную директорию
	tempDir := t.TempDir()

	// Путь к тестовому .env файлу
	envFilePath := filepath.Join(tempDir, ".env")

	// Записываем в .env значения по умолчанию
	dotEnvContent := []byte(`
	DB_DRIVER_NAME=mysql
	DB_DSN=user=mysql password=1234 dbname=mydb sslmode=disable
	`)
	err := os.WriteFile(envFilePath, dotEnvContent, 0644)
	assert.NoError(t, err, "Не удалось создать тестовый .env файл")

	// Устанавливаем другие значения в окружении
	err = os.Setenv("DB_DRIVER_NAME", "postgres")
	require.NoError(t, err)
	err = os.Setenv("DB_DSN", "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable")
	require.NoError(t, err)

	cfg := common.GetConfig(envFilePath)
	assert.Equal(t, "postgres", cfg.DbDriverName)
	assert.Equal(t, "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable", cfg.Dsn)
}

func Test_GetConfig_LoadsFromDotEnv_WhenNoConflictingEnvVars(t *testing.T) {
	// Создаем временную директорию
	tempDir := t.TempDir()

	// Путь к тестовому .env файлу
	envFilePath := filepath.Join(tempDir, ".env")

	// Записываем в него корректные значения
	dotEnvContent := []byte(`
	DB_DRIVER_NAME=postgres
	DB_DSN=host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable
	`)
	err := os.WriteFile(envFilePath, dotEnvContent, 0644)
	assert.NoError(t, err, "Не удалось создать тестовый .env файл")

	// Убеждаемся, что переменные окружения не установлены
	err = os.Unsetenv("DB_DRIVER_NAME")
	require.NoError(t, err)
	err = os.Unsetenv("DB_DSN")
	require.NoError(t, err)

	// Переходим в временную директорию, чтобы относительный путь ".env" работал
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Получаем конфиг
	cfg := common.GetConfig(".env")

	// Проверяем, что значения взяты из .env
	assert.Equal(t, "postgres", cfg.DbDriverName)
	assert.Contains(t, cfg.Dsn, "dbname=mydb")
	assert.Contains(t, cfg.Dsn, "user=postgres")
	assert.Contains(t, cfg.Dsn, "password=1234")
}

func Test_ConnectDb_WithInvalidConfig_ShouldError(t *testing.T) {
	// Создаем временную директорию
	tempDir := t.TempDir()

	// Путь к тестовому .env файлу
	envFilePath := filepath.Join(tempDir, ".env")

	// Записываем в него заведомо неверные данные (например, логин или пароль)
	dotEnvContent := []byte(`
	DB_DRIVER_NAME=postgres
	DB_DSN=host=localhost port=5432 user=wronguser password=wrongpass dbname=postgres sslmode=disable
	`)
	err := os.WriteFile(envFilePath, dotEnvContent, 0644)
	require.NoError(t, err)

	// Получаем конфиг из файла
	cfg := common.GetConfig(envFilePath)

	// Вызываем подключение — должно вызвать Error
	db, err := database.ConnectDbWithCfg(cfg)
	assert.Nil(t, db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

func Test_ConnectDb_WithValidConfig_ShouldSucceed(t *testing.T) {
	err := os.Unsetenv("DB_DRIVER_NAME")
	require.NoError(t, err)
	err = os.Unsetenv("DB_DSN")
	require.NoError(t, err)

	envFilePath := filepath.Join("..", ".env")

	cfg := common.GetConfig(envFilePath)

	// Подключаемся к БД
	db, err := database.ConnectDbWithCfg(cfg)

	assert.NoError(t, err, "Ошибка подключения к БД")
	assert.NotNil(t, db, "Ожидается непустое соединение с БД")

	// Проверка соединения простым SQL-запросом
	if db != nil {
		var version string
		err = db.Get(&version, "SELECT version()")
		assert.NoError(t, err, "Ошибка при выполнении SELECT version()")
		assert.Contains(t, version, "PostgreSQL 17.5", "Ожидается, что БД — это PostgreSQL")
		fmt.Println("Database version:", version)
	}
}
