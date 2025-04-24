package order

import (
	"context"
	"errors"
	"fmt"
)

// ErrDuplicateOrderID indicates that an order with the given ID already exists.
var ErrDuplicateOrderID = errors.New("order with this ID already exists")

// OrderService defines methods for managing orders.
type OrderService interface {
	// CreateOrder creates a new order.
	CreateOrder(ctx context.Context, order *Order) error
	// GetOrderByID retrieves an order by its ID.
	GetOrderByID(ctx context.Context, id string) (*Order, error)
}

type orderService struct {
	repo OrderRepository
}

// NewOrderService creates a new OrderService.
func NewOrderService(repo OrderRepository) *orderService {
	return &orderService{repo: repo}
}

// CreateOrder validates and creates a new order.
func (s *orderService) CreateOrder(ctx context.Context, order *Order) error {
	if order.Total < 0 {
		return fmt.Errorf("total must be non-negative, got %f", order.Total)
	}

	if order.Status == "" {
		return fmt.Errorf("status is required")
	}

	exists, err := s.repo.ExistsByID(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("failed to check order existence: %w", err)
	}

	if exists {
		return ErrDuplicateOrderID
	}

	return s.repo.Create(ctx, order)
}

// GetOrderByID retrieves an order by its ID.
func (s *orderService) GetOrderByID(ctx context.Context, id string) (*Order, error) {
	return s.repo.GetByID(ctx, id)
}
