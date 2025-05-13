package db

import (
	"context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/internal/config"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

func New(cfg config.PostgresConfig) (*Postgres, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=order_service", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

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

	log.Info().Msg("Connected to PostgreSQL")
	return &Postgres{Pool: dbPool}, nil
}

func (p *Postgres) Close() {
	p.Pool.Close()
	log.Info().Msg("Database connection closed")
}

func applyMigrations(dbPool *pgxpool.Pool, postgresCfg config.PostgresConfig) error {
	// Преобразуем pgxpool.Pool в sql.DB для миграций
	sqlDB := stdlib.OpenDBFromPool(dbPool)
	defer sqlDB.Close()

	// Проверка подключения
	err := sqlDB.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database for migrations: %w", err)
	}

	poolCfg := dbPool.Config()
	dsn := fmt.Sprintf("pgx5://%s:%s@%s:%d/%s?sslmode=%s",
		poolCfg.ConnConfig.User,
		poolCfg.ConnConfig.Password,
		poolCfg.ConnConfig.Host,
		poolCfg.ConnConfig.Port,
		poolCfg.ConnConfig.Database,
		postgresCfg.SSLMode,
	)

	m, err := migrate.New("file://"+postgresCfg.MigrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize migration instance: %w", err)
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		log.Info().Msg("No new migrations to apply")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	log.Info().Msg("New migrations applied successfully")

	return nil
}
