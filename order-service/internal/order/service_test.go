package order_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/internal/order"
)

type mockOrderRepository struct {
	createFunc     func(ctx context.Context, order *order.Order) error
	getByIDFunc    func(ctx context.Context, id string) (*order.Order, error)
	existsByIDFunc func(ctx context.Context, id string) (bool, error)
}

func (m *mockOrderRepository) Create(ctx context.Context, order *order.Order) error {
	return m.createFunc(ctx, order)
}

func (m *mockOrderRepository) GetByID(ctx context.Context, id string) (*order.Order, error) {
	return m.getByIDFunc(ctx, id)
}

func (m *mockOrderRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	return m.existsByIDFunc(ctx, id)
}

func TestOrderService_CreateOrder(t *testing.T) {
	tests := []struct {
		name           string
		order          *order.Order
		existsByIDFunc func(ctx context.Context, id string) (bool, error)
		createFunc     func(ctx context.Context, order *order.Order) error
		wantErr        bool
		wantErrIs      error
		wantErrMsg     string
	}{
		{
			name: "negative_total",
			order: &order.Order{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				UserID:    "123e4567-e89b-12d3-a456-426614174000",
				Total:     -10,
				Status:    "created",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			existsByIDFunc: func(ctx context.Context, id string) (bool, error) { return false, nil },
			createFunc:     func(ctx context.Context, order *order.Order) error { return nil },
			wantErr:        true,
			wantErrMsg:     "total must be non-negative, got -10.000000",
		},
		{
			name: "empty_status",
			order: &order.Order{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				UserID:    "123e4567-e89b-12d3-a456-426614174000",
				Total:     100,
				Status:    "",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			existsByIDFunc: func(ctx context.Context, id string) (bool, error) { return false, nil },
			createFunc:     func(ctx context.Context, order *order.Order) error { return nil },
			wantErr:        true,
			wantErrMsg:     "status is required",
		},
		{
			name: "duplicate_id",
			order: &order.Order{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				UserID:    "123e4567-e89b-12d3-a456-426614174000",
				Total:     100,
				Status:    "created",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			existsByIDFunc: func(ctx context.Context, id string) (bool, error) { return true, nil },
			createFunc:     func(ctx context.Context, order *order.Order) error { return nil },
			wantErr:        true,
			wantErrIs:      order.ErrDuplicateOrderID,
			wantErrMsg:     "order with this ID already exists",
		},
		{
			name: "successful_creation",
			order: &order.Order{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				UserID:    "123e4567-e89b-12d3-a456-426614174000",
				Total:     100,
				Status:    "created",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			existsByIDFunc: func(ctx context.Context, id string) (bool, error) { return false, nil },
			createFunc:     func(ctx context.Context, order *order.Order) error { return nil },
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockOrderRepository{
				createFunc:     tt.createFunc,
				getByIDFunc:    func(ctx context.Context, id string) (*order.Order, error) { return nil, nil },
				existsByIDFunc: tt.existsByIDFunc,
			}
			svc := order.NewOrderService(mockRepo)
			err := svc.CreateOrder(context.Background(), tt.order)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrIs != nil {
					assert.True(t, errors.Is(err, tt.wantErrIs))
				}
				if tt.wantErrMsg != "" {
					assert.Equal(t, tt.wantErrMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrderService_GetOrderByID(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		getByIDFunc func(ctx context.Context, id string) (*order.Order, error)
		expected    *order.Order
		wantErr     bool
	}{
		{
			name: "success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			getByIDFunc: func(ctx context.Context, id string) (*order.Order, error) {
				return &order.Order{
					ID:        "550e8400-e29b-41d4-a716-446655440000",
					UserID:    "123e4567-e89b-12d3-a456-426614174000",
					Total:     100.50,
					Status:    "created",
					CreatedAt: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC),
				}, nil
			},
			expected: &order.Order{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				UserID:    "123e4567-e89b-12d3-a456-426614174000",
				Total:     100.50,
				Status:    "created",
				CreatedAt: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "not_found",
			id:   "999e8400-e29b-41d4-a716-446655440000",
			getByIDFunc: func(ctx context.Context, id string) (*order.Order, error) {
				return nil, nil
			},
			expected: nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockOrderRepository{
				createFunc:     func(ctx context.Context, order *order.Order) error { return nil },
				getByIDFunc:    tt.getByIDFunc,
				existsByIDFunc: func(ctx context.Context, id string) (bool, error) { return false, nil },
			}
			svc := order.NewOrderService(mockRepo)
			ord, err := svc.GetOrderByID(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, ord)
			}
		})
	}
}
