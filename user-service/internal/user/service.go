package user

import "context"

type Service interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo Repository
}

func (s *service) CreateUser(ctx context.Context, user *User) (*User, error) {
	// Логика создания пользователя (например, хеширование пароля)
	return s.repo.Create(ctx, user)
}

func (s *service) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *service) Update(ctx context.Context, user *User) (*User, error) {
	return s.repo.Update(ctx, user)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
