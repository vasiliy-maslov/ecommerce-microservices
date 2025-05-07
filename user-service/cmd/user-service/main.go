package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/config"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/db"
	userHttp "github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/handler/http"
	userService "github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/user"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	log.Logger = log.With().Str("service", "user-service").Logger()

	log.Info().Msg("Starting user-service...")

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to load config: %v", err)
	}
	log.Debug().Interface("config_loaded", cfg).Msg("Configuration loaded")
	dbPool, err := db.New(cfg.Postgres)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to connect to database: %v", err)
	}
	log.Info().Msg("Successfully conected to database")

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
			log.Fatal().Err(err).Msgf("Could not listen on %s: %v\n", cfg.App.Port, err)
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh

	log.Info().Msg("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msgf("Server Shutdown Failed:%+v", err)
	}

	dbPool.Close()

	log.Info().Msg("HTTP server stopped.")

	log.Info().Msg("User-service stopped gracefully.")
}
