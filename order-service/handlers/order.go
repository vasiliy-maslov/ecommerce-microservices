package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/entities"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/services"
)

// OrderHandler handles HTTP requests for orders.
type OrderHandler struct {
	svc services.OrderService
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(svc services.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// CreateOrder handles the creation of a new order.
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var order entities.Order

	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if err := h.svc.CreateOrder(ctx, &order); err != nil {
		if errors.Is(err, services.ErrDuplicateOrderID) {
			http.Error(w, "order with this ID already exists", http.StatusConflict)
			return
		}
		log.Printf("Failed to create order: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(&order); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "invalid json", http.StatusInternalServerError)
		return
	}
}

// GetOrderByID handles retrieving an order by its ID.
func (h *OrderHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	order, err := h.svc.GetOrderByID(ctx, id)
	if err != nil {
		log.Printf("Failed to get order by id: %v", err)
		http.Error(w, "failed to get order", http.StatusInternalServerError)
		return
	}
	if order == nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(&order); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "invalid json", http.StatusInternalServerError)
		return
	}
}
