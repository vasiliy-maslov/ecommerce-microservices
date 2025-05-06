package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/user"
)

type CreateUserRequest struct {
	FirstName string `json:"first_name" validate:"required,min=2"`
	LastName  string `json:"last_name" validate:"required,min=2"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
}

type UpdateUserRequest struct {
	FirstName string  `json:"first_name" validate:"required,min=2"`
	LastName  string  `json:"last_name" validate:"required,min=2"`
	Email     string  `json:"email" validate:"required,email"`
	Password  *string `json:"password,omitempty" validate:"omitempty,min=8"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserHandler struct {
	service  user.Service
	validate *validator.Validate
}

func NewUserHandler(service user.Service) *UserHandler {
	validate := validator.New()
	return &UserHandler{
		service:  service,
		validate: validate,
	}
}

func (h *UserHandler) RegisterRoutes(router chi.Router) {
	router.Post("/users", h.handleCreateUser)
	router.Get("/users/{id}", h.handleGetUserByID)
	router.Get("/users/email/{email}", h.handleGetUserByEmail)
	router.Put("/users/{id}", h.handleUpdateUser)
	router.Delete("/users/{id}", h.handleDeleteUser)
}

func (h *UserHandler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var requestPayload CreateUserRequest

	decoder := json.NewDecoder(r.Body)

	decoder.DisallowUnknownFields()

	err := decoder.Decode(&requestPayload)
	if err != nil {
		log.Printf("ERROR: Failed to decode request bode: %v", err)
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request payload %v", err))
		return
	}

	err = h.validate.Struct(requestPayload)
	if err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if ok {
			respondWithError(w, http.StatusBadRequest, formatValidationErrors(validationErrors))
		} else {
			log.Printf("ERROR: Unexpected error type during validation: %T, %v", err, err)
			respondWithError(w, http.StatusInternalServerError, "Internal validation error")
		}
		return
	}

	domainUser := user.User{
		FirstName:    requestPayload.FirstName,
		LastName:     requestPayload.LastName,
		Email:        requestPayload.Email,
		PasswordHash: requestPayload.Password,
	}

	createdUser, err := h.service.CreateUser(r.Context(), &domainUser)
	if err != nil {
		log.Printf("ERROR: Failed to create user via service: %v", err)

		statusCode := mapErrorToStatusCode(err)

		var clientMessage string

		if errors.Is(err, user.ErrEmailExists) {
			clientMessage = "Email already exists"
		} else {
			clientMessage = "Failed to create user"
		}

		respondWithError(w, statusCode, clientMessage)
		return
	}

	responsePayload := UserResponse{
		ID:        createdUser.ID,
		FirstName: createdUser.FirstName,
		LastName:  createdUser.LastName,
		Email:     createdUser.Email,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
	}

	respondWithJSON(w, http.StatusCreated, responsePayload)
}

func (h *UserHandler) handleGetUserByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	userID, err := uuid.FromString(idParam)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid id parameter: %v", err))
		return
	}

	foundUser, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR: Failed to get user by id via service: %v", err)

		statusCode := mapErrorToStatusCode(err)

		var clientMessage string

		if errors.Is(err, user.ErrNotFound) {
			clientMessage = "User not found"
		} else {
			clientMessage = "Failed to get user by id"
		}

		respondWithError(w, statusCode, clientMessage)
		return
	}

	responsePayload := UserResponse{
		ID:        foundUser.ID,
		FirstName: foundUser.FirstName,
		LastName:  foundUser.LastName,
		Email:     foundUser.Email,
		CreatedAt: foundUser.CreatedAt,
		UpdatedAt: foundUser.UpdatedAt,
	}

	respondWithJSON(w, http.StatusOK, responsePayload)
}

func (h *UserHandler) handleGetUserByEmail(w http.ResponseWriter, r *http.Request) {
	emailParam := chi.URLParam(r, "email")
	if emailParam == "" {
		respondWithError(w, http.StatusBadRequest, "Email parameter cannot be empty")
		return
	}

	foundUser, err := h.service.GetUserByEmail(r.Context(), emailParam)
	if err != nil {
		log.Printf("ERROR: Failed to get user by email via service: %v", err)

		statusCode := mapErrorToStatusCode(err)

		var clientMessage string

		if errors.Is(err, user.ErrNotFound) {
			clientMessage = "User not found"
		} else {
			clientMessage = "Failed to get user by email"
		}

		respondWithError(w, statusCode, clientMessage)
		return
	}

	responsePayload := UserResponse{
		ID:        foundUser.ID,
		FirstName: foundUser.FirstName,
		LastName:  foundUser.LastName,
		Email:     foundUser.Email,
		CreatedAt: foundUser.CreatedAt,
		UpdatedAt: foundUser.UpdatedAt,
	}

	respondWithJSON(w, http.StatusOK, responsePayload)
}

func (h *UserHandler) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	userID, err := uuid.FromString(idParam)
	if err != nil {
		log.Printf("ERROR: Failed to parse id from param: %v", err)

		respondWithError(w, http.StatusBadRequest, "Invalid id parameter")
		return
	}

	var requestPayload UpdateUserRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&requestPayload)
	if err != nil {
		log.Printf("ERROR: Failed to decode user: %v", err)

		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err = h.validate.Struct(requestPayload)
	if err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if ok {
			respondWithError(w, http.StatusBadRequest, formatValidationErrors(validationErrors))
		} else {
			log.Printf("ERROR: Unexpected error type during validation: %T, %v", err, err)
			respondWithError(w, http.StatusInternalServerError, "Internal validation error")
		}
		return
	}

	domainUser := user.User{
		ID:        userID,
		FirstName: requestPayload.FirstName,
		LastName:  requestPayload.LastName,
		Email:     requestPayload.Email,
	}

	if requestPayload.Password != nil {
		domainUser.PasswordHash = *requestPayload.Password
	}

	err = h.service.UpdateUser(r.Context(), &domainUser)
	if err != nil {
		log.Printf("ERROR: Failed to update user via service: %v", err)

		statusCode := mapErrorToStatusCode(err)

		var clientMessage string

		if errors.Is(err, user.ErrNotFound) {
			clientMessage = "User not found"
		} else if errors.Is(err, user.ErrEmailExists) {
			clientMessage = "Email already exists"
		} else {
			clientMessage = "Failed to update user"
		}

		respondWithError(w, statusCode, clientMessage)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	userID, err := uuid.FromString(idParam)
	if err != nil {
		log.Printf("ERROR: Failed to parse id from param: %v", err)

		respondWithError(w, http.StatusBadRequest, "Invalid id parameter")
		return
	}

	err = h.service.DeleteUser(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR: Failed to delete user via service: %v", err)

		statusCode := mapErrorToStatusCode(err)

		var clientMessage string

		if errors.Is(err, user.ErrNotFound) {
			clientMessage = "User not found"
		} else {
			clientMessage = "Failed to delete user"
		}

		respondWithError(w, statusCode, clientMessage)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
