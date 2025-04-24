package db

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Connect(cfg Config) (*sqlx.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Connected to PostgreSQL")

	err = applyMigrations(db, cfg.DBName)
	if err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return db, nil
}

func applyMigrations(db *sqlx.DB, dbName string) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get current file path: %w", err)
	}

	currentDir := filepath.Dir(filename)
	rootDir := filepath.Dir(filepath.Dir(currentDir))
	migrationsPath := filepath.Join(rootDir, "migrations")

	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, dbName, driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migration instance: %w", err)
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		log.Println("No new migrations to apply")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	log.Println("New migrations applied successfully")

	return nil
}
