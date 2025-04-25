package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxConns        int32         // Максимальное количество соединений
	MinConns        int32         // Минимальное количество соединений
	MaxConnLifetime time.Duration // Максимальное время жизни соединения
	MigrationsPath  string
}

type Postgres struct {
	Pool *pgxpool.Pool
}

func New(cfg Config) (*Postgres, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Настройка пула соединений (опционально)
	config.MaxConns = cfg.MaxConns
	config.MinConns = cfg.MinConns
	config.MaxConnLifetime = cfg.MaxConnLifetime

	dbPool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	err = applyMigrations(dbPool, cfg.MigrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("Connected to PostgreSQL")
	return &Postgres{Pool: dbPool}, nil
}

func (p *Postgres) Close() {
	p.Pool.Close()
	log.Println("Database connection closed")
}

func applyMigrations(dbPool *pgxpool.Pool, migrationsPath string) error {
	// Преобразуем pgxpool.Pool в sql.DB для миграций
	sqlDB := stdlib.OpenDBFromPool(dbPool)
	defer sqlDB.Close()

	// Проверка подключения
	err := sqlDB.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database for migrations: %w", err)
	}

	m, err := migrate.New("file://"+migrationsPath, "pgx5://"+dbPool.Config().ConnString())
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
