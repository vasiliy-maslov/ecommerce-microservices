package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/config"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/db"
	userHttp "github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/handler/http"
	userService "github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/user"
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

	userRepository := userService.NewRepository(dbPool.Pool)
	userSvc := userService.NewService(userRepository)
	userHandler := userHttp.NewUserHandler(userSvc)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	userHandler.RegisterRoutes(router)

	server := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Starting HTTP server on port %s", cfg.App.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Could not listen on %s: %v\n", cfg.App.Port, err)
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	dbPool.Close()

	log.Println("HTTP server stopped.")

	log.Println("User-service stopped gracefully.")
}
