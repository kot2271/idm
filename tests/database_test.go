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
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	// Путь к существующему .env файлу в корне проекта
	// envFilePath := filepath.Join("..", ".env")
	envFilePath := filepath.Join("..", ".env_not_exists")

	// Ожидаем панику из-за валидации
	assert.Panics(t, func() {
		common.GetConfig(envFilePath)
	}, "Должна быть паника из-за отсутствия обязательных полей")
}

func Test_GetConfig_NoEnvFile_WithPanicMessage(t *testing.T) {
	// Удаляем переменные окружения для чистоты теста
	envVars := []string{"DB_DRIVER_NAME", "DB_DSN", "APP_NAME", "APP_VERSION", "LOG_LEVEL", "LOG_DEVELOP_MODE"}
	for _, envVar := range envVars {
		err := os.Unsetenv(envVar)
		require.NoError(t, err)
	}

	// Путь к несуществующему .env файлу
	envFilePath := filepath.Join("..", ".env_not_exists")

	// Ожидаем панику и проверяем содержимое сообщения
	defer func() {
		if r := recover(); r != nil {
			panicMsg, ok := r.(string)
			assert.True(t, ok, "Паника должна содержать строку")
			assert.Contains(t, panicMsg, "config validation error", "Сообщение паники должно содержать информацию о валидации")
		}
	}()

	// Этот вызов должен вызвать панику
	assert.Panics(t, func() {
		common.GetConfig(envFilePath)
	})
}

func Test_GetConfig_NoVarsInEnvAndDotEnv(t *testing.T) {
	// Убедимся, что переменные окружения не заданы
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	// Создаем временную директорию
	tempDir := t.TempDir()

	// Путь к фейковому .env файлу
	envFilePath := filepath.Join(tempDir, ".env")

	// Создаем пустой .env файл (или без нужных переменных)
	err := os.WriteFile(envFilePath, []byte(""), 0644)
	assert.NoError(t, err, "Не удалось создать временный .env файл")

	// Ожидаем панику из-за валидации
	assert.Panics(t, func() {
		common.GetConfig(envFilePath)
	}, "Должна быть паника из-за отсутствия обязательных полей")
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
	err = os.Setenv("APP_NAME", "test-app")
	require.NoError(t, err)
	err = os.Setenv("APP_VERSION", "1.0.0")
	require.NoError(t, err)
	err = os.Setenv("LOG_LEVEL", "DEBUG")
	require.NoError(t, err)
	err = os.Setenv("LOG_DEVELOP_MODE", "true")
	require.NoError(t, err)
	err = os.Setenv("SSL_SERT", "ssl.cert")
	require.NoError(t, err)
	err = os.Setenv("SSL_KEY", "ssl.key")
	require.NoError(t, err)

	// Вызываем GetConfig с путём к тестовому .env
	cfg := common.GetConfig(envFilePath)
	assert.NotEmpty(t, cfg)

	// Проверяем, что значения взяты из переменных окружения
	assert.Equal(t, "postgres", cfg.DbDriverName)
	assert.Equal(t, "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable", cfg.Dsn)
	assert.Equal(t, "test-app", cfg.AppName)
	assert.Equal(t, "1.0.0", cfg.AppVersion)
	assert.Equal(t, "DEBUG", cfg.LogLevel)
	assert.Equal(t, true, cfg.LogDevelopMode)
	assert.Equal(t, "ssl.cert", cfg.SslSert)
	assert.Equal(t, "ssl.key", cfg.SslKey)

	// Очистка переменных окружения после теста
	defer func() {
		cleanupEnvVars(t)
	}()
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
	APP_NAME=dotenv-app
	APP_VERSION=2.0.0
	LOG_LEVEL=DEBUG
	LOG_DEVELOP_MODE=true
	SSL_SERT=ssl.cert
	SSL_KEY=ssl.key
	`)
	err := os.WriteFile(envFilePath, dotEnvContent, 0644)
	assert.NoError(t, err, "Не удалось создать тестовый .env файл")

	// Устанавливаем другие значения в окружении
	err = os.Setenv("DB_DRIVER_NAME", "postgres")
	require.NoError(t, err)
	err = os.Setenv("DB_DSN", "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable")
	require.NoError(t, err)
	err = os.Setenv("APP_NAME", "env-app")
	require.NoError(t, err)
	err = os.Setenv("APP_VERSION", "3.0.0")
	require.NoError(t, err)
	err = os.Setenv("LOG_LEVEL", "DEBUG")
	require.NoError(t, err)
	err = os.Setenv("LOG_DEVELOP_MODE", "true")
	require.NoError(t, err)
	err = os.Setenv("SSL_SERT", "ssl.cert")
	require.NoError(t, err)
	err = os.Setenv("SSL_KEY", "ssl.key")
	require.NoError(t, err)

	cfg := common.GetConfig(envFilePath)
	assert.NotEmpty(t, cfg)

	assert.Equal(t, "postgres", cfg.DbDriverName)
	assert.Equal(t, "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable", cfg.Dsn)
	assert.Equal(t, "env-app", cfg.AppName)
	assert.Equal(t, "3.0.0", cfg.AppVersion)
	assert.Equal(t, "DEBUG", cfg.LogLevel)
	assert.Equal(t, true, cfg.LogDevelopMode)
	assert.Equal(t, "ssl.cert", cfg.SslSert)
	assert.Equal(t, "ssl.key", cfg.SslKey)

	defer func() {
		cleanupEnvVars(t)
	}()

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
	APP_NAME=dotenv-app
	APP_VERSION=1.5.0
	LOG_LEVEL=DEBUG
	LOG_DEVELOP_MODE=true
	SSL_SERT=ssl.cert
	SSL_KEY=ssl.key
	`)
	err := os.WriteFile(envFilePath, dotEnvContent, 0644)
	assert.NoError(t, err, "Не удалось создать тестовый .env файл")

	// Убеждаемся, что переменные окружения не установлены
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	// Переходим в временную директорию, чтобы относительный путь ".env" работал
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Получаем конфиг
	cfg := common.GetConfig(".env")
	assert.NotEmpty(t, cfg)

	// Проверяем, что значения взяты из .env
	assert.Equal(t, "postgres", cfg.DbDriverName)
	assert.Contains(t, cfg.Dsn, "dbname=mydb")
	assert.Contains(t, cfg.Dsn, "user=postgres")
	assert.Contains(t, cfg.Dsn, "password=1234")
	assert.Equal(t, "dotenv-app", cfg.AppName)
	assert.Equal(t, "1.5.0", cfg.AppVersion)
	assert.Equal(t, "DEBUG", cfg.LogLevel)
	assert.Equal(t, true, cfg.LogDevelopMode)
	assert.Equal(t, "ssl.cert", cfg.SslSert)
	assert.Equal(t, "ssl.key", cfg.SslKey)
}

func Test_ConnectDb_WithInvalidConfig_ShouldError(t *testing.T) {
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	// Создаем временную директорию
	tempDir := t.TempDir()

	// Путь к тестовому .env файлу
	envFilePath := filepath.Join(tempDir, ".env")

	// Записываем в него заведомо неверные данные (например, логин или пароль)
	dotEnvContent := []byte(`
	DB_DRIVER_NAME=postgres
	DB_DSN=host=localhost port=5432 user=wronguser password=wrongpass dbname=postgres sslmode=disable
	APP_NAME=test-app
	APP_VERSION=1.0.0
	LOG_LEVEL=DEBUG
	LOG_DEVELOP_MODE=true
	SSL_SERT=ssl.cert
	SSL_KEY=ssl.key
	`)
	err := os.WriteFile(envFilePath, dotEnvContent, 0644)
	require.NoError(t, err)

	// Получаем конфиг из файла
	cfg := common.GetConfig(envFilePath)
	assert.NotEmpty(t, cfg)

	// Вызываем подключение — должно вызвать Error
	db, err := database.ConnectDbWithCfg(cfg)
	assert.Nil(t, db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

func Test_ConnectDb_WithValidConfig_ShouldSucceed(t *testing.T) {
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	envFilePath := filepath.Join("..", ".env")

	cfg := common.GetConfig(envFilePath)
	assert.NotEmpty(t, cfg)

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

func TestGetConfig_ValidConfig(t *testing.T) {
	// временный .env файл с валидными данными
	envContent :=
		`
	DB_DRIVER_NAME=postgres
	DB_DSN=host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable
	APP_NAME=test_app
	APP_VERSION=1.0.0
	LOG_LEVEL=DEBUG
	LOG_DEVELOP_MODE=true
	SSL_SERT=/path/to/ssl.cert
	SSL_KEY=/path/to/ssl.key
	`

	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	require.NoError(t, os.WriteFile(envFile, []byte(envContent), 0644))

	cfg := common.GetConfig(envFile)

	assert.Equal(t, "postgres", cfg.DbDriverName)
	assert.Equal(t, "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable", cfg.Dsn)
	assert.Equal(t, "test_app", cfg.AppName)
	assert.Equal(t, "1.0.0", cfg.AppVersion)
	assert.Equal(t, "DEBUG", cfg.LogLevel)
	assert.True(t, cfg.LogDevelopMode)
	assert.Equal(t, "/path/to/ssl.cert", cfg.SslSert)
	assert.Equal(t, "/path/to/ssl.key", cfg.SslKey)
}

func TestGetConfig_MissingSslCert_ShouldPanic(t *testing.T) {
	// Очищаем переменные окружения перед тестом
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	// Принудительно очищаем SSL переменные окружения
	err := os.Unsetenv("SSL_SERT")
	require.NoError(t, err)
	err = os.Unsetenv("SSL_KEY")
	require.NoError(t, err)

	// .env файл без SSL_SERT
	envContent :=
		`
	DB_DRIVER_NAME=postgres
	DB_DSN=host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable
	APP_NAME=test_app
	APP_VERSION=1.0.0
	LOG_LEVEL=DEBUG
	LOG_DEVELOP_MODE=true
	SSL_KEY=/path/to/ssl.key
	`

	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	require.NoError(t, os.WriteFile(envFile, []byte(envContent), 0644))

	// Проверяем, что функция паникует при отсутствии SSL_SERT
	assert.Panics(t, func() {
		common.GetConfig(envFile)
	}, "GetConfig should panic when SSL_SERT is missing")
}

func TestGetConfig_MissingSslKey_ShouldPanic(t *testing.T) {
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	// .env файл без SSL_KEY
	envContent :=
		`
	DB_DRIVER_NAME=postgres
	DB_DSN=host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable
	APP_NAME=test_app
	APP_VERSION=1.0.0
	LOG_LEVEL=DEBUG
	LOG_DEVELOP_MODE=true
	SSL_SERT=/path/to/ssl.cert`

	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	require.NoError(t, os.WriteFile(envFile, []byte(envContent), 0644))

	// Проверяем, что функция паникует при отсутствии SSL_KEY
	assert.Panics(t, func() {
		common.GetConfig(envFile)
	}, "GetConfig should panic when SSL_KEY is missing")
}

func TestGetConfig_MissingBothSslFields_ShouldPanic(t *testing.T) {
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	// .env файл без SSL полей
	envContent :=
		`
	DB_DRIVER_NAME=postgres
	DB_DSN=host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable
	APP_NAME=test_app
	APP_VERSION=1.0.0
	LOG_LEVEL=DEBUG
	LOG_DEVELOP_MODE=true
	`

	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	require.NoError(t, os.WriteFile(envFile, []byte(envContent), 0644))

	// Проверяем, что функция паникует при отсутствии обоих SSL полей
	assert.Panics(t, func() {
		common.GetConfig(envFile)
	}, "GetConfig should panic when both SSL_SERT and SSL_KEY are missing")
}

func TestGetConfig_EmptySslFields_ShouldPanic(t *testing.T) {
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	// Создаем .env файл с пустыми SSL полями
	envContent := `
	DB_DRIVER_NAME=postgres
	DB_DSN=host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable
	APP_NAME=test_app
	APP_VERSION=1.0.0
	LOG_LEVEL=DEBUG
	LOG_DEVELOP_MODE=true
	SSL_SERT=
	SSL_KEY=
	`

	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	require.NoError(t, os.WriteFile(envFile, []byte(envContent), 0644))

	// Проверяем, что функция паникует при пустых SSL полях
	assert.Panics(t, func() {
		common.GetConfig(envFile)
	}, "GetConfig should panic when SSL fields are empty")
}

func TestGetConfig_FromEnvironmentVariables(t *testing.T) {
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	err := os.Setenv("DB_DRIVER_NAME", "postgres")
	require.NoError(t, err)
	err = os.Setenv("DB_DSN", "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable")
	require.NoError(t, err)
	err = os.Setenv("APP_NAME", "env_test_app")
	require.NoError(t, err)
	err = os.Setenv("APP_VERSION", "2.0.0")
	require.NoError(t, err)
	err = os.Setenv("LOG_LEVEL", "INFO")
	require.NoError(t, err)
	err = os.Setenv("LOG_DEVELOP_MODE", "true")
	require.NoError(t, err)
	err = os.Setenv("SSL_SERT", "/env/path/to/ssl.cert")
	require.NoError(t, err)
	err = os.Setenv("SSL_KEY", "/env/path/to/ssl.key")
	require.NoError(t, err)

	cfg := common.GetConfig("nonexistent.env")

	assert.Equal(t, "postgres", cfg.DbDriverName)
	assert.Equal(t, "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable", cfg.Dsn)
	assert.Equal(t, "env_test_app", cfg.AppName)
	assert.Equal(t, "2.0.0", cfg.AppVersion)
	assert.Equal(t, "INFO", cfg.LogLevel)
	assert.Equal(t, true, cfg.LogDevelopMode)
	assert.Equal(t, "/env/path/to/ssl.cert", cfg.SslSert)
	assert.Equal(t, "/env/path/to/ssl.key", cfg.SslKey)
}

func TestGetConfig_MissingEnvSslCert_ShouldPanic(t *testing.T) {
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	err := os.Setenv("DB_DRIVER_NAME", "postgres")
	require.NoError(t, err)
	err = os.Setenv("DB_DSN", "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable")
	require.NoError(t, err)
	err = os.Setenv("APP_NAME", "env_test_app")
	require.NoError(t, err)
	err = os.Setenv("APP_VERSION", "2.0.0")
	require.NoError(t, err)
	err = os.Setenv("LOG_LEVEL", "INFO")
	require.NoError(t, err)
	err = os.Setenv("LOG_DEVELOP_MODE", "true")
	require.NoError(t, err)
	err = os.Setenv("SSL_KEY", "/env/path/to/ssl.key")
	require.NoError(t, err)

	// Проверяем панику при отсутствии SSL_SERT в env
	assert.Panics(t, func() {
		common.GetConfig("nonexistent.env")
	}, "GetConfig should panic when SSL_SERT env var is missing")
}

func TestGetConfig_MissingEnvSslKey_ShouldPanic(t *testing.T) {
	t.Cleanup(func() {
		cleanupEnvVars(t)
	})

	err := os.Setenv("DB_DRIVER_NAME", "postgres")
	require.NoError(t, err)
	err = os.Setenv("DB_DSN", "host=localhost port=5432 user=postgres password=1234 dbname=mydb sslmode=disable")
	require.NoError(t, err)
	err = os.Setenv("APP_NAME", "env_test_app")
	require.NoError(t, err)
	err = os.Setenv("APP_VERSION", "2.0.0")
	require.NoError(t, err)
	err = os.Setenv("LOG_LEVEL", "INFO")
	require.NoError(t, err)
	err = os.Setenv("LOG_DEVELOP_MODE", "true")
	require.NoError(t, err)
	err = os.Setenv("SSL_SERT", "/env/path/to/ssl.cert")
	require.NoError(t, err)

	// Проверяем панику при отсутствии SSL_KEY в env
	assert.Panics(t, func() {
		common.GetConfig("nonexistent.env")
	}, "GetConfig should panic when SSL_KEY env var is missing")
}

// очищает все переменные окружения
func cleanupEnvVars(t *testing.T) {
	envVars := []string{
		"DB_DRIVER_NAME",
		"DB_DSN",
		"APP_NAME",
		"APP_VERSION",
		"LOG_LEVEL",
		"LOG_DEVELOP_MODE",
		"SSL_SERT",
		"SSL_KEY",
	}

	for _, envVar := range envVars {
		err := os.Unsetenv(envVar)
		require.NoError(t, err)
	}
}
