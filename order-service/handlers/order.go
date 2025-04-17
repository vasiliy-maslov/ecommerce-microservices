package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/entities"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/repositories"
)

type OrderHandler struct {
	repo repositories.OrderRepository
}

func NewOrderHandler(repo repositories.OrderRepository) *OrderHandler {
	return &OrderHandler{repo: repo}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var order entities.Order

	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	if err := h.repo.Create(ctx, &order); err != nil {
		log.Printf("Failed to create order: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(&order); err != nil {
		http.Error(w, "invalid json", http.StatusInternalServerError)
		return
	}
}

func (h *OrderHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	order, err := h.repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("Failed to get order by id: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(&order); err != nil {
		http.Error(w, "invalid json", http.StatusInternalServerError)
		return
	}
}
