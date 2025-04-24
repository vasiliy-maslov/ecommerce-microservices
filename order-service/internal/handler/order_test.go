package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/internal/order"
)

type mockOrderService struct {
	CreateOrderFunc  func(ctx context.Context, ord *order.Order) error
	GetOrderByIDFunc func(ctx context.Context, id string) (*order.Order, error)
}

func (m *mockOrderService) CreateOrder(ctx context.Context, order *order.Order) error {
	return m.CreateOrderFunc(ctx, order)
}

func (m *mockOrderService) GetOrderByID(ctx context.Context, id string) (*order.Order, error) {
	return m.GetOrderByIDFunc(ctx, id)
}

func TestOrderHandler_CreateOrder(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		createOrder    func(ctx context.Context, ord *order.Order) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			body: `{
				"id": "550e8400-e29b-41d4-a716-446655440000",
				"user_id": "123e4567-e89b-12d3-a456-426614174000",
				"total": 100.50,
				"status": "created",
				"created_at": "2025-04-16T12:00:00Z",
				"updated_at": "2025-04-16T12:00:00Z"
			}`,
			createOrder: func(ctx context.Context, ord *order.Order) error {
				return nil
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"id":"550e8400-e29b-41d4-a716-446655440000","user_id":"123e4567-e89b-12d3-a456-426614174000","total":100.5,"status":"created","created_at":"2025-04-16T12:00:00Z","updated_at":"2025-04-16T12:00:00Z"}` + "\n",
		},
		{
			name: "duplicate_id",
			body: `{
				"id": "550e8400-e29b-41d4-a716-446655440000",
				"user_id": "123e4567-e89b-12d3-a456-426614174000",
				"total": 100.50,
				"status": "created",
				"created_at": "2025-04-16T12:00:00Z",
				"updated_at": "2025-04-16T12:00:00Z"
			}`,
			createOrder: func(ctx context.Context, ord *order.Order) error {
				return order.ErrDuplicateOrderID
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   "order with this ID already exists\n",
		},
		{
			name:           "invalid_json",
			body:           `{invalid json}`,
			createOrder:    func(ctx context.Context, order *order.Order) error { return nil },
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid request body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockOrderService{
				CreateOrderFunc: tt.createOrder,
				GetOrderByIDFunc: func(ctx context.Context, id string) (*order.Order, error) {
					return nil, nil
				},
			}

			handler := NewOrderHandler(mockSvc)
			r := chi.NewRouter()
			r.Post("/orders", handler.CreateOrder)

			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestOrderHandler_GetOrderByID(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		getOrderByID   func(ctx context.Context, id string) (*order.Order, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			getOrderByID: func(ctx context.Context, id string) (*order.Order, error) {
				return &order.Order{
					ID:        "550e8400-e29b-41d4-a716-446655440000",
					UserID:    "123e4567-e89b-12d3-a456-426614174000",
					Total:     100.50,
					Status:    "created",
					CreatedAt: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC),
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"550e8400-e29b-41d4-a716-446655440000","user_id":"123e4567-e89b-12d3-a456-426614174000","total":100.5,"status":"created","created_at":"2025-04-16T12:00:00Z","updated_at":"2025-04-16T12:00:00Z"}` + "\n",
		},
		{
			name: "not_found",
			id:   "999e8400-e29b-41d4-a716-446655440000",
			getOrderByID: func(ctx context.Context, id string) (*order.Order, error) {
				return nil, nil
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "order not found\n",
		},
		{
			name: "empty_id",
			id:   "", // Пустой id
			getOrderByID: func(ctx context.Context, id string) (*order.Order, error) {
				return nil, nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "id is required\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockOrderService{
				CreateOrderFunc:  func(ctx context.Context, ord *order.Order) error { return nil },
				GetOrderByIDFunc: tt.getOrderByID,
			}

			handler := NewOrderHandler(mockSvc)

			// Создаём запрос
			req := httptest.NewRequest(http.MethodGet, "/orders/"+tt.id, nil)
			// Вручную задаём параметр id в контексте
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			// Напрямую вызываем хендлер, минуя маршрутизацию chi
			handler.GetOrderByID(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}
