package services_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/entities"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/services"
)

type mockOrderRepository struct {
	createFunc  func(ctx context.Context, order *entities.Order) error
	getByIDFunc func(ctx context.Context, id string) (*entities.Order, error)
}

func (m *mockOrderRepository) Create(ctx context.Context, order *entities.Order) error {
	return m.createFunc(ctx, order)
}

func (m *mockOrderRepository) GetByID(ctx context.Context, id string) (*entities.Order, error) {
	return m.getByIDFunc(ctx, id)
}

func newTestOrder(total float64, status string) *entities.Order {
	return &entities.Order{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		UserID:    "123e4567-e89b-12d3-a456-426614174000",
		Total:     total,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestOrderService_CreateOrder(t *testing.T) {
	mockRepo := &mockOrderRepository{
		createFunc:  func(ctx context.Context, order *entities.Order) error { return nil },
		getByIDFunc: func(ctx context.Context, id string) (*entities.Order, error) { return nil, nil },
	}

	svc := services.NewOrderService(mockRepo)
	ctx := context.Background()

	// Тест 1: total < 0
	order := newTestOrder(-10, "created")
	err := svc.CreateOrder(ctx, order)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "total must be non-negative")

	// Тест 2: status пустой
	order = newTestOrder(100, "")
	err = svc.CreateOrder(ctx, order)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status is required")

	// Тест 3: успешное создание
	order = newTestOrder(100, "created")
	err = svc.CreateOrder(ctx, order)
	assert.NoError(t, err)
}

func TestOrderService_GetOrderByID(t *testing.T) {
	// Создай мок-репозиторий
	mockRepo := &mockOrderRepository{
		createFunc: func(ctx context.Context, order *entities.Order) error {
			return nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*entities.Order, error) {
			return nil, nil
		},
	}
	svc := services.NewOrderService(mockRepo)
	ctx := context.Background()

	// Тест 1: заказ существует
	mockRepo.getByIDFunc = func(ctx context.Context, id string) (*entities.Order, error) {
		return &entities.Order{
			ID:        id,
			UserID:    "123e4567-e89b-12d3-a456-426614174000",
			Total:     100,
			Status:    "created",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}
	order, err := svc.GetOrderByID(ctx, "some-id")
	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, order.ID, "some-id")

	// Тест 2: заказ не существует
	mockRepo.getByIDFunc = func(ctx context.Context, id string) (*entities.Order, error) {
		return nil, sql.ErrNoRows
	}

	order, err = svc.GetOrderByID(ctx, "unknown-id")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, sql.ErrNoRows))
	assert.Nil(t, order)
}
