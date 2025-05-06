package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
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

func formatValidationErrors(errs validator.ValidationErrors) string {
	var messages []string
	for _, err := range errs {
		var msg string
		switch err.Tag() {
		case "required":
			msg = fmt.Sprintf("Field '%s' is required", err.Field())
		case "min":
			msg = fmt.Sprintf("Field '%s' must be at least %s characters long", err.Field(), err.Param())
		case "email":
			msg = fmt.Sprintf("Field '%s' must be a valid email address", err.Field())
		// Добавьте другие case для других тегов, которые вы используете
		default:
			// Стандартное сообщение, если не знаем тег
			msg = fmt.Sprintf("Field '%s' failed validation on '%s'", err.Field(), err.Tag())
		}
		messages = append(messages, msg)
	}
	return strings.Join(messages, "; ") // Объединяем все сообщения
}
