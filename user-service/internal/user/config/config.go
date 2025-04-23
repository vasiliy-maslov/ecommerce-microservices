package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	} `yaml:"app"`

	Postgres struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"postgres"`
}

func Load(path string) *Config {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("failed open config file: %v", err)
	}

	var config Config

	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		log.Fatalf("invalid config file: %v", err)
	}

	return &config
}
