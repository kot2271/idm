package database

import (
	"fmt"
	"idm/inner/common"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// Получить конфиг и подключиться с ним к базе данных
func ConnectDb() (*sqlx.DB, error) {
	cfg := common.GetConfig(".env")
	return ConnectDbWithCfg(cfg)
}

// Подключиться к базе данных с переданным конфигом
func ConnectDbWithCfg(cfg common.Config) (*sqlx.DB, error) {
	logger := common.NewLogger(cfg)
	defer func() {
		if err := logger.Sync(); err != nil {
			// В случае ошибки при синхронизации логгера выводим в stderr
			// т.к. сам логгер может быть недоступен
			fmt.Printf("Failed to sync logger: %v\n", err)
		}
	}()

	db, err := sqlx.Connect(cfg.DbDriverName, cfg.Dsn)
	if err != nil {
		logger.Error("Failed to connect to database",
			zap.String("driver", cfg.DbDriverName),
			zap.Error(err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logger.Info("Database connection established successfully",
		zap.String("driver", cfg.DbDriverName))

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(1 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	logger.Debug("Database connection pool configured",
		zap.Int("maxIdleConns", 5),
		zap.Int("maxOpenConns", 20),
		zap.Duration("connMaxLifetime", 1*time.Minute),
		zap.Duration("connMaxIdleTime", 10*time.Minute))

	return db, nil
}
