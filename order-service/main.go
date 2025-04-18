package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/db"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/handlers"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/repositories"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/services"
)

func main() {
	log.Println("Order service starting...")

	err := godotenv.Load("order-service/.env")
	if err != nil && err != os.ErrNotExist {
		log.Fatalf("failed to load .env: %v", err)
	}

	dbCfg := db.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	if dbCfg.Host == "" {
		log.Fatalf("DB_HOST is required")
	}
	if dbCfg.Port == "" {
		log.Fatalf("DB_PORT is required")
	}
	if dbCfg.User == "" {
		log.Fatalf("DB_USER is required")
	}
	if dbCfg.Password == "" {
		log.Fatalf("DB_PASSWORD is required")
	}
	if dbCfg.DBName == "" {
		log.Fatalf("DB_NAME is required")
	}

	dbConn, err := db.Connect(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	repo := repositories.NewPostgresOrderRepository(dbConn)
	svc := services.NewOrderService(repo)
	handler := handlers.NewOrderHandler(svc)

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.Post("/orders", handler.CreateOrder)
	r.Get("/orders/{id}", handler.GetOrderByID)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Starting server on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown failed: %v", err)
	}
	log.Println("Server stopped")
}
