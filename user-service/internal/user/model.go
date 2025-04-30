package user

import (
	"time"

	"github.com/gofrs/uuid"
)

// User представляет структуру данных пользователя.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`                 // ID пользователя
	FirstName    string    `json:"first_name" db:"first_name"` // Имя
	LastName     string    `json:"last_name" db:"last_name"`   // Фамилия
	Email        string    `json:"email" db:"email"`           // Электронная почта
	PasswordHash string    `json:"-" db:"password_hash"`       // Пароль (не возвращаем в ответах)
	CreatedAt    time.Time `json:"created_at" db:"created_at"` // Время создания
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"` // Время обновления
}
