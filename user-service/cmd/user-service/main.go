package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/config"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/db"
)

func main() {
	log.Println("Starting user-service...")

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbPool, err := db.New(cfg.Postgres)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh

	log.Println("Shutting down...")
	dbPool.Close()
	log.Println("User-service stopped gracefully.")
}
