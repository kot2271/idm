package database

import (
	"fmt"
	"idm/inner/common"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Получить конфиг и подключиться с ним к базе данных
func ConnectDb() (*sqlx.DB, error) {
	cfg := common.GetConfig(".env")
	return ConnectDbWithCfg(cfg)
}

// Подключиться к базе данных с переданным конфигом
func ConnectDbWithCfg(cfg common.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect(cfg.DbDriverName, cfg.Dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	fmt.Println("Database connection established successfully")

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(1 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	return db, nil
}
