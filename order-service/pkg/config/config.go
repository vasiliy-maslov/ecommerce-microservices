package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	App struct {
		Port string
	}
	Postgres struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
		SSLMode  string
	}
}

func Load(path string) (*Config, error) {
	if path != "" {
		err := godotenv.Load(path)
		if err != nil && err != os.ErrNotExist {
			return nil, fmt.Errorf("failed to load .env: %w", err)
		}
	}

	cfg := &Config{}
	cfg.App.Port = os.Getenv("APP_PORT")
	if cfg.App.Port == "" {
		cfg.App.Port = "8080"
	}

	cfg.Postgres.Host = os.Getenv("DB_HOST")
	if cfg.Postgres.Host == "" {
		log.Fatalf("DB_HOST is required")
	}
	cfg.Postgres.Port = os.Getenv("DB_PORT")
	if cfg.Postgres.Port == "" {
		log.Fatalf("DB_PORT is required")
	}
	cfg.Postgres.User = os.Getenv("DB_USER")
	if cfg.Postgres.User == "" {
		log.Fatalf("DB_USER is required")
	}
	cfg.Postgres.Password = os.Getenv("DB_PASSWORD")
	if cfg.Postgres.Password == "" {
		log.Fatalf("DB_PASSWORD is required")
	}
	cfg.Postgres.DBName = os.Getenv("DB_NAME")
	if cfg.Postgres.DBName == "" {
		log.Fatalf("DB_NAME is required")
	}
	cfg.Postgres.SSLMode = os.Getenv("DB_SSLMODE")
	if cfg.Postgres.SSLMode == "" {
		cfg.Postgres.SSLMode = "disable"
	}

	return cfg, nil
}
