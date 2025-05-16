package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/user"
)

type ValidationErrorResponse struct {
	Error   string            `json:"error"`   // Общее сообщение
	Details map[string]string `json:"details"` // Детали по полям
}

// respondWithError отправляет JSON ошибку
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON отправляет JSON ответ
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Type("payload_type", payload).Msg("ERROR: Failed to marshal JSON response")
		w.WriteHeader(http.StatusInternalServerError)
		if _, writeErr := w.Write([]byte(`{"error":"Failed to marshal JSON response"}`)); writeErr != nil {
			log.Warn().Err(writeErr).Msg("Failed to write fallback error response")
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(response); err != nil {
		log.Error().Err(err).Int("status_code", code).Msg("Failed to write JSON response")
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

func formatValidationErrors(errs validator.ValidationErrors) map[string]string {
	errorDetails := make(map[string]string)
	for _, err := range errs {
		var msg string
		field := err.Field()
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
		errorDetails[field] = msg
	}
	return errorDetails
}
