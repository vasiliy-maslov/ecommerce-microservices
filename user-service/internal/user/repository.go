package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound              = errors.New("user not found")
	ErrEmailExists           = errors.New("email already exists")
	ErrCannotUpdateAdminUser = errors.New("can not update admin")
)

// Репозиторий для работы с пользователями.
type Repository interface {
	Create(ctx context.Context, user *User) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type DB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Реализация репозитория для работы с БД.
type repository struct {
	db DB // Предположим, что у нас есть интерфейс DB для работы с базой
}

// В этой функции подключаемся к БД и возвращаем репозиторий.
func NewRepository(db DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user *User) (uuid.UUID, error) {
	query := `
		INSERT INTO user_service.users (
			id,
			first_name, 
			last_name,
			email,
			password_hash
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	row := r.db.QueryRow(ctx, query,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Email,
		user.PasswordHash,
	)

	var createdID uuid.UUID
	err := row.Scan(&createdID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return uuid.Nil, ErrEmailExists
			}
		}
		return uuid.Nil, fmt.Errorf("failed to create user and scan returned id: %w", err)
	}

	return createdID, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT
			id,
			first_name,
			last_name,
			email,
			password_hash,
			created_at,
			updated_at
		FROM user_service.users
		WHERE id = $1
	`

	row := r.db.QueryRow(ctx, query, id)
	var user User
	err := row.Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to scan user by id %s: %w", id.String(), err)
	}

	return &user, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT
			id,
			first_name,
			last_name,
			email,
			password_hash,
			created_at,
			updated_at
		FROM user_service.users
		WHERE email = $1
	`

	row := r.db.QueryRow(ctx, query, email)
	var user User
	err := row.Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to scan user by email %s: %w", email, err)
	}

	return &user, nil
}

func (r *repository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE user_service.users
		SET 
			first_name = $1,
			last_name = $2,
			email = $3,
			password_hash = $4,
			updated_at = $5
		WHERE
			id = $6
	`

	tag, err := r.db.Exec(ctx, query,
		user.FirstName,
		user.LastName,
		user.Email,
		user.PasswordHash,
		time.Now(),
		user.ID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrEmailExists
			}
		}

		return fmt.Errorf("failed to update user by id %s: %w", user.ID, err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM user_service.users WHERE id = $1
	`

	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user by id %s: %w", id.String(), err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
