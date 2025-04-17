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
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/db"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/handlers"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/repositories"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/services"
)

func main() {
	log.Println("Order service starting...")

	dbCfg := db.Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "123456",
		DBName:   "orders",
		SSLMode:  "disable",
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
