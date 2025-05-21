package database

import (
	"fmt"
	"idm/inner/common"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Временная переменная, ссылается на подключение к базе данных
var DB *sqlx.DB

// Получить конфиг и подключиться с ним к базе данных
func ConnectDb() *sqlx.DB {
	cfg := common.GetConfig(".env")
	return ConnectDbWithCfg(cfg)
}

// Подключиться к базе данных с переданным конфигом
func ConnectDbWithCfg(cfg common.Config) *sqlx.DB {
	DB = sqlx.MustConnect(cfg.DbDriverName, cfg.Dsn)
	fmt.Println("Database connection established successfully")

	DB.SetMaxIdleConns(5)
	DB.SetMaxOpenConns(20)
	DB.SetConnMaxLifetime(1 * time.Minute)
	DB.SetConnMaxIdleTime(10 * time.Minute)
	return DB
}
