package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
)

var allowedTransitions = map[OrderStatus]map[OrderStatus]bool{
	StatusNew: {
		StatusProcessing: true,
		StatusCancelled:  true,
	},
	StatusProcessing: {
		StatusPaid:      true,
		StatusShipped:   true,
		StatusCancelled: true,
	},
	StatusPaid: {
		StatusShipped:   true,
		StatusCancelled: true,
	},
	StatusShipped: {
		StatusDelivered: true,
		StatusCancelled: true,
	},
	StatusDelivered: {},
	StatusCancelled: {},
}

var (
	ErrStatusAlreadySet        = errors.New("status is already set to the desired value")
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
)

type Service interface {
	CreateOrder(ctx context.Context, orderInput *Order) (*Order, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (*Order, error)
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, newStatus OrderStatus) error
}

type service struct {
	orderRepo Repository // Наша зависимость от репозитория заказов
	// productRepo ProductRepository // Пример будущей зависимости
}

func NewService(orderRepo Repository) Service {
	return &service{
		orderRepo: orderRepo,
	}
}

func (s *service) CreateOrder(ctx context.Context, orderInput *Order) (*Order, error) {
	totalAmount := 0.0

	if len(orderInput.OrderItems) == 0 {
		log.Warn().Msg("service: attempt to create order with no items")
		return nil, errors.New("service: order must contain at least one item")
	}

	orderInput.ID = uuid.Nil

	for i := range orderInput.OrderItems {
		item := &orderInput.OrderItems[i]

		if item.Quantity <= 0 {
			return nil, fmt.Errorf("service: order item quantity for product %s must be greater than zero", item.ProductID)
		}

		if item.PricePerUnit < 0 {
			return nil, fmt.Errorf("service: order item price per unit for product %s cannot be negative", item.ProductID)
		}

		if item.ProductID == uuid.Nil {
			return nil, errors.New("service: product id in order item cannot be nil")
		}

		item.ID = uuid.Nil
		item.OrderID = uuid.Nil

		totalAmount += float64(item.Quantity) * item.PricePerUnit
	}

	orderInput.Status = StatusNew
	orderInput.TotalAmount = totalAmount

	_, err := s.orderRepo.CreateOrder(ctx, orderInput)
	if err != nil {
		log.Error().Err(err).Msg("service: failed to create order in repository")
		return nil, fmt.Errorf("service: failed to create order: %w", err)
	}

	log.Info().Stringer("order_id", orderInput.ID).Stringer("user_id", orderInput.UserID).Msg("Service: Order created successfully")

	return orderInput, nil
}

func (s *service) GetOrderByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	order, err := s.orderRepo.GetOrderByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			log.Warn().Err(err).Stringer("order_id", id).Msg("service: order not found by id")
			return nil, ErrOrderNotFound
		}

		log.Error().Err(err).Msg("service: failed to fetch order by id in repository")
		return nil, fmt.Errorf("service: failed to fetch order by id: %w", err)
	}

	return order, nil
}

func (s *service) GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]Order, error) {
	orders, err := s.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		log.Error().Err(err).Stringer("user_id", userID).Msg("service: failed to fetch user orders in repository")
		return nil, fmt.Errorf("service: failed to fetch user orders: %w", err)
	}

	return orders, nil
}

func (s *service) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, newStatus OrderStatus) error {
	// 1. (Проверка входного newStatus - для твоего типа OrderStatus не так критично)

	// 2. Получение текущего заказа
	currentOrder, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			log.Warn().Err(err).Stringer("order_id", orderID).Stringer("new_status", newStatus).Msg("service: order not found, cannot update status")
			return ErrOrderNotFound
		}
		log.Error().Err(err).Stringer("order_id", orderID).Msg("service: failed to get order for status update")
		return fmt.Errorf("service: failed to get order for status update: %w", err)
	}

	// 3. Проверка на изменение статуса
	if currentOrder.Status == newStatus {
		log.Info().Stringer("order_id", orderID).Stringer("status", newStatus).Msg("service: order status is already the same, no update needed")
		return nil
	}

	// 4. Валидация перехода статусов (State Machine)
	transitionsForCurrentStatus, ok := allowedTransitions[currentOrder.Status]
	if !ok || !transitionsForCurrentStatus[newStatus] { // Объединенная проверка
		// !ok означает, что для currentOrder.Status вообще нет правил (ошибка конфигурации).
		// !transitionsForCurrentStatus[newStatus] означает, что конкретный переход к newStatus не разрешен.
		logMessage := "service: invalid status transition attempt"
		// Можно добавить более детальное сообщение, если !ok
		if !ok {
			logMessage = fmt.Sprintf("service: no transition rules defined for current status %s", currentOrder.Status)
		}

		log.Warn().
			Stringer("order_id", currentOrder.ID).
			Stringer("current_status", currentOrder.Status).
			Stringer("new_status", newStatus).
			Msg(logMessage)
		return fmt.Errorf("service: invalid status transition from %s to %s (or no rules for %s)", currentOrder.Status, newStatus, currentOrder.Status)
	}

	// 5. Обновление статуса в репозитории
	err = s.orderRepo.UpdateOrderStatus(ctx, orderID, newStatus)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			log.Warn().Err(err).Stringer("order_id", orderID).Stringer("new_status", newStatus).Msg("service: order not found during final update status call (race condition?)")
			return ErrOrderNotFound
		}
		log.Error().Err(err).Stringer("order_id", orderID).Stringer("new_status", newStatus).Msg("service: failed to update order status in repository")
		return fmt.Errorf("service: failed to update order status: %w", err)
	}

	// 6. Логирование успеха
	log.Info().Stringer("order_id", orderID).Stringer("old_status", currentOrder.Status).Stringer("new_status", newStatus).Msg("service: order status updated successfully")
	return nil
}
