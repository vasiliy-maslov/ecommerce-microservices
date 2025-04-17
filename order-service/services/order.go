package services

import (
	"context"
	"fmt"

	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/entities"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/repositories"
)

type OrderService interface {
	CreateOrder(ctx context.Context, order *entities.Order) error
	GetOrderByID(ctx context.Context, id string) (*entities.Order, error)
}

type orderService struct {
	repo repositories.OrderRepository
}

func NewOrderService(repo repositories.OrderRepository) *orderService {
	return &orderService{repo: repo}
}

func (s *orderService) CreateOrder(ctx context.Context, order *entities.Order) error {
	if order.Total < 0 {
		return fmt.Errorf("total must be non-negative")
	}

	if order.Status == "" {
		return fmt.Errorf("status is required")
	}

	return s.repo.Create(ctx, order)
}

func (s *orderService) GetOrderByID(ctx context.Context, id string) (*entities.Order, error) {
	return s.repo.GetByID(ctx, id)
}
