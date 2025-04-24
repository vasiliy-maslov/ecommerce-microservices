package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/config"
)

func main() {
	cfg := config.Load("config/config.yaml")
	fmt.Printf("Loaded config: %+v\n", cfg)

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	log.Println("Server started successful")

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("failed starting server: %v", err)
	}

}
