package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type AppConfig struct {
	Port string
}

type PostgresConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MigrationsPath  string
}

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
}

func NewConfig(path string) (*Config, error) {
	cfg := &Config{}
	var err error

	// Загрузка конфигурации приложения
	cfg.App.Port = os.Getenv("APP_PORT")
	if cfg.App.Port == "" {
		return nil, errors.New("APP_PORT environment variable not set")
	}

	// Загрузка конфигурации PostgreSQL
	cfg.Postgres.Host = os.Getenv("DB_HOST")
	if cfg.Postgres.Host == "" {
		return nil, errors.New("DB_HOST environment variable not set")
	}

	cfg.Postgres.Port = os.Getenv("DB_PORT")
	if cfg.Postgres.Port == "" {
		return nil, errors.New("DB_PORT environment variable not set")
	}

	cfg.Postgres.User = os.Getenv("DB_USER")
	if cfg.Postgres.User == "" {
		return nil, errors.New("DB_USER environment variable not set")
	}

	cfg.Postgres.Password = os.Getenv("DB_PASSWORD")
	if cfg.Postgres.Password == "" {
		return nil, errors.New("DB_PASSWORD environment variable not set") // В реальных проектах пароль может быть опциональным или обрабатываться иначе
	}

	cfg.Postgres.DBName = os.Getenv("DB_NAME")
	if cfg.Postgres.DBName == "" {
		return nil, errors.New("DB_NAME environment variable not set")
	}

	cfg.Postgres.SSLMode = os.Getenv("DB_SSLMODE")
	if cfg.Postgres.SSLMode == "" {
		cfg.Postgres.SSLMode = "disable"
	}

	// Преобразование MaxConns из строки в int32
	maxConnsStr := os.Getenv("DB_MAX_CONNS")
	if maxConnsStr == "" {
		// Если переменная не установлена, используем значение по умолчанию
		cfg.Postgres.MaxConns = 20
	} else {
		// Если переменная установлена, парсим ее
		maxConnsInt, err := strconv.ParseInt(maxConnsStr, 10, 32)
		if err != nil {
			// Если парсинг не удался, возвращаем ошибку
			return nil, fmt.Errorf("failed to parse DB_MAX_CONNS '%s': %w", maxConnsStr, err)
		}
		// Если парсинг успешен, используем спарсенное значение
		cfg.Postgres.MaxConns = int32(maxConnsInt)
	}

	// Аналогично для MinConns
	minConnsStr := os.Getenv("DB_MIN_CONNS")
	if minConnsStr == "" {
		cfg.Postgres.MinConns = 2
	} else {
		minConnsInt, err := strconv.ParseInt(minConnsStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DB_MIN_CONNS '%s': %w", minConnsStr, err)
		}
		cfg.Postgres.MinConns = int32(minConnsInt)
	}

	// Аналогично для MaxConnLifetime
	maxConnLifetimeStr := os.Getenv("DB_MAX_CONN_LIFETIME")
	if maxConnLifetimeStr == "" {
		cfg.Postgres.MaxConnLifetime = 30 * time.Minute
	} else {
		cfg.Postgres.MaxConnLifetime, err = time.ParseDuration(maxConnLifetimeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DB_MAX_CONN_LIFETIME '%s': %w", maxConnLifetimeStr, err)
		}
	}

	cfg.Postgres.MigrationsPath = os.Getenv("MIGRATIONS_PATH")
	if cfg.Postgres.MigrationsPath == "" {
		cfg.Postgres.MigrationsPath = "/app/migrations"
	}

	return cfg, nil
}
