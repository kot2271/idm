package tests

import (
	"fmt"
	"log"
	"os"
	"testing"

	"idm/inner/common"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "testuser"
	password = "testpass"
	dbname   = "testdb"
)

var DB *sqlx.DB
var config common.Config

func init() {
	var err error
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	DB, err = sqlx.Connect("postgres", connStr)

	dsnStr := fmt.Sprintf("%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	config = common.Config{
		DbDriverName:   "postgres",
		Dsn:            dsnStr,
		AppName:        "test_app",
		AppVersion:     "1.0.0",
		LogLevel:       "DEBUG",
		LogDevelopMode: true,
	}
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	applyMigrations()
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func applyMigrations() {
	_, err := DB.Exec(`
        CREATE TABLE IF NOT EXISTS role (
            id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
            name TEXT NOT NULL,
            description TEXT,
            status BOOLEAN DEFAULT TRUE,
            parent_id BIGINT REFERENCES role(id) ON DELETE SET NULL,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS employee (
            id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
            name TEXT NOT NULL,
            email TEXT UNIQUE NOT NULL,
            position TEXT,
            department TEXT,
            role_id BIGINT REFERENCES role(id),
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );
    `)
	if err != nil {
		log.Fatalf("Migration failed: %v\n", err)
	}
}

func clearTables() {
	if DB == nil {
		log.Fatal("Database connection is nil")
	}
	_, err := DB.Exec("DELETE FROM employee")
	if err != nil {
		log.Fatalf("Failed to clear employee table: %v", err)
	}
	_, err = DB.Exec("DELETE FROM role")
	if err != nil {
		log.Fatalf("Failed to clear role table: %v", err)
	}
}
