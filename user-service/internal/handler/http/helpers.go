package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/user"
)

// respondWithError отправляет JSON ошибку
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON отправляет JSON ответ
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal JSON response: %v", payload)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"Failed to marshal JSON response"}`)) // Простой ответ
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(response); err != nil {
		log.Printf("ERROR: Failed to write JSON response: %v", err)
	}
}

func mapErrorToStatusCode(err error) int {
	switch {
	case errors.Is(err, user.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, user.ErrEmailExists):
		return http.StatusConflict
	case errors.Is(err, user.ErrCannotUpdateAdminUser):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
