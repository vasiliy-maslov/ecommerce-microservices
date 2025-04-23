package user

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	service Service
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func NewRouter(service Service) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	h := NewHandler(service)

	// Создание нового пользователя
	r.Post("/users", h.CreateUser)
	r.Get("/users/{id}", h.GetUserByID)
	r.Get("/users", h.GetByEmail)
	r.Put("/users/{id}", h.Update)
	r.Delete("/users/{id}", h.Delete)

	return r
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("invalid json: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	createdUser, err := h.service.CreateUser(r.Context(), &user)
	if err != nil {
		log.Printf("failed to create user: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, createdUser)
}

func (h *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if id == "" {
		http.Error(w, "required id", http.StatusBadRequest)
		return
	}

	i, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("failed to convert id to int: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user, err := h.service.GetUserByID(r.Context(), int64(i))
	if err != nil {
		log.Printf("failed to get user by id: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func (h *Handler) GetByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email query param is required", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetByEmail(r.Context(), email)
	if err != nil {
		log.Printf("failed to get user by email: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("invalid json: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	u, err := h.service.Update(r.Context(), &user)
	if err != nil {
		log.Printf("failed to update user: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, u)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if id == "" {
		http.Error(w, "required id", http.StatusBadRequest)
		return
	}

	i, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("failed to convert id to int: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.service.Delete(r.Context(), int64(i))
	if err != nil {
		log.Printf("failed to delete user: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
