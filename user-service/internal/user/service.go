package user

import (
	"context"
	"errors"
	"fmt" // Понадобится для оборачивания ошибок
	"log" // Для логирования хеширования

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateUser(ctx context.Context, user *User) (*User, error) {
	if user.PasswordHash == "" {
		return nil, errors.New("password cannot be empty")
	}
	hashPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: failed to generate hash password: %v", err)
		return nil, fmt.Errorf("internal error hashing password: %w", err)
	}
	user.PasswordHash = string(hashPasswordBytes)

	createdID, err := s.repo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			return nil, ErrEmailExists
		}
		log.Printf("ERROR: failed to create user in repository: %v", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	user.ID = createdID

	return user, nil
}

func (s *service) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}

		log.Printf("ERROR: failed to get user by id in repository: %v", err)
		return nil, fmt.Errorf("failed to get user by id '%s': %w", id, err)
	}

	return user, nil
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}

		log.Printf("ERROR: failed to get user by email in repository: %v", err)
		return nil, fmt.Errorf("failed to get user by email '%s': %w", email, err)
	}

	return user, nil
}

func (s *service) UpdateUser(ctx context.Context, user *User) error {
	if user.PasswordHash != "" {
		newPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("ERROR: failed to generate hash password: %v", err)
			return fmt.Errorf("failed to generate hash password: %w", err)
		}

		user.PasswordHash = string(newPassword)
	}

	err := s.repo.Update(ctx, user)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			return ErrEmailExists
		}

		log.Printf("ERROR: failed to update user: %v", err)
		return fmt.Errorf("failed to update user by id '%s': %w", user.ID.String(), err)
	}

	return nil
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}

		log.Printf("ERROR: failed to delete user: %v", err)
		return fmt.Errorf("failed to delete user by id '%s': %w", id, err)
	}

	return nil
}
