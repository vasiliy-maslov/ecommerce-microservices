package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var (
	ErrOrderNotFound   = errors.New("order not found")
	ErrProductNotFound = errors.New("product not found")
)

type Repository interface {
	CreateOrder(ctx context.Context, order *Order) (uuid.UUID, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (*Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, newStatus OrderStatus) error
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]Order, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateOrder(ctx context.Context, orderInput *Order) (orderID uuid.UUID, err error) {
	finalOrderID := orderInput.ID // Используем ID из orderInput если он там есть (например, из тестов)
	if finalOrderID == uuid.Nil { // Если ID не предоставлен (типичный случай от сервиса)
		genID, genErr := uuid.NewV4()
		if genErr != nil {
			// Эта ошибка маловероятна с gofrs/uuid, но для полноты
			log.Error().Err(genErr).Msg("repository: failed to generate order ID")
			return uuid.Nil, fmt.Errorf("repository: failed to generate order ID: %w", genErr)
		}
		finalOrderID = genID
	}
	// Записываем ID обратно в orderInput, чтобы вызывающий код (сервис) увидел его, если он был сгенерирован здесь.
	// Хотя сервис все равно будет использовать возвращаемое значение orderID.
	// Но если сервис передал orderInput дальше, это может быть полезно.
	// ВАЖНО: orderInput - это указатель, так что это изменение будет видно вызывающему коду.
	orderInput.ID = finalOrderID // Обновляем ID в исходном объекте

	tx, beginErr := r.db.Begin(ctx)
	if beginErr != nil {
		// ... (логирование)
		return uuid.Nil, fmt.Errorf("repository: failed to begin transaction: %w", beginErr)
	}
	// Используем defer с recover, как договорились
	defer func() {
		if p := recover(); p != nil {
			log.Error().Interface("panic_value", p).Stringer("order_id_attempted", finalOrderID).Msg("Panic recovered during CreateOrder, rolling back")
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Error().Err(rbErr).Stringer("order_id_attempted", finalOrderID).Msg("Failed to rollback transaction after panic")
			}
			panic(p) // Перепаниковать
		} else if err != nil {
			log.Warn().Err(err).Stringer("order_id_attempted", finalOrderID).Msg("Transaction for CreateOrder failed, rolling back")
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Error().Err(rbErr).Stringer("order_id_attempted", finalOrderID).Msg("Failed to rollback transaction")
			}
		} else {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				log.Error().Err(commitErr).Stringer("order_id", finalOrderID).Msg("Failed to commit transaction")
				err = fmt.Errorf("repository: failed to commit transaction: %w", commitErr)
				// Не нужно сбрасывать finalOrderID, так как err != nil и defer это обработает
			}
		}
	}()

	// 1. Вставляем основной заказ
	orderCreatedAt := time.Now().UTC()
	orderUpdatedAt := orderCreatedAt // При создании равны

	queryOrder := `
		INSERT INTO order_service.orders (id, user_id, status, total_amount, shipping_address_text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.Exec(ctx, queryOrder,
		finalOrderID,
		orderInput.UserID,
		string(orderInput.Status),
		orderInput.TotalAmount,
		orderInput.ShippingAddressText,
		orderCreatedAt,
		orderUpdatedAt,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("repository: failed to insert order: %w", err)
	}
	// Обновляем orderInput с серверными значениями CreatedAt/UpdatedAt
	orderInput.CreatedAt = orderCreatedAt
	orderInput.UpdatedAt = orderUpdatedAt

	// 2. Вставляем позиции заказа
	for i := range orderInput.OrderItems {
		itemInput := &orderInput.OrderItems[i] // Работаем с указателем на элемент слайса

		itemID, genErr := uuid.NewV4() // Генерируем ID для позиции здесь
		if genErr != nil {
			return uuid.Nil, fmt.Errorf("repository: failed to generate order item ID: %w", genErr)
		}
		itemInput.ID = itemID // Устанавливаем сгенерированный ID в объект

		itemCreatedAt := time.Now().UTC()
		itemUpdatedAt := itemCreatedAt

		queryItem := `
			INSERT INTO order_service.order_items (id, order_id, product_id, quantity, price_per_unit, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		_, err = tx.Exec(ctx, queryItem,
			itemInput.ID,
			finalOrderID, // ID родительского заказа
			itemInput.ProductID,
			itemInput.Quantity,
			itemInput.PricePerUnit,
			itemCreatedAt,
			itemUpdatedAt,
		)
		if err != nil {
			return uuid.Nil, fmt.Errorf("repository: failed to insert order item for order %s: %w", finalOrderID, err)
		}
		// Обновляем itemInput с серверными значениями
		itemInput.CreatedAt = itemCreatedAt
		itemInput.UpdatedAt = itemUpdatedAt
		itemInput.OrderID = finalOrderID // Убедимся, что OrderID установлен в объекте OrderItem
	}
	return finalOrderID, nil // err будет nil если commit успешен
}

func (r *postgresRepository) GetOrderByID(ctx context.Context, orderID uuid.UUID) (*Order, error) {
	queryOrder := `
		SELECT id, user_id, status, total_amount, shipping_address_text, created_at, updated_at 
		FROM order_service.orders
		WHERE id = $1
	`

	var order Order
	err := r.db.QueryRow(ctx, queryOrder, orderID).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&order.ShippingAddressText,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}

		return nil, fmt.Errorf("repository: failed to select order by id %s: %w", orderID, err)
	}

	queryOrderItems := `
		SELECT id, order_id, product_id, quantity, price_per_unit, created_at, updated_at
		FROM order_service.order_items
		WHERE order_id = $1
	`

	rows, err := r.db.Query(ctx, queryOrderItems, orderID)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to query order items for order id %s: %w", orderID, err)
	}

	orderItems := make([]OrderItem, 0)
	for rows.Next() {
		var orderItem OrderItem
		err := rows.Scan(
			&orderItem.ID,
			&orderItem.OrderID,
			&orderItem.ProductID,
			&orderItem.Quantity,
			&orderItem.PricePerUnit,
			&orderItem.CreatedAt,
			&orderItem.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: failed to scan order item for order id %s: %w", orderID, err)
		}
		orderItems = append(orderItems, orderItem)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: error iterating order items for order id %s: %w", orderID, err)
	}

	order.OrderItems = orderItems

	return &order, nil
}

func (r *postgresRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, newStatus OrderStatus) error {
	query := `
		UPDATE order_service.orders
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	cmdTag, err := r.db.Exec(ctx, query,
		string(newStatus),
		time.Now(),
		orderID,
	)
	if err != nil {
		log.Error().Err(err).Stringer("order_id", orderID).Str("new_status", string(newStatus)).Msg("repository: failed to update order status")
		return fmt.Errorf("repository: failed to update order status %s: %w", orderID, err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Warn().Stringer("order_id", orderID).Str("new_status", string(newStatus)).Msg("repository: order not found for status update")
		return ErrOrderNotFound
	}

	return nil
}

func (r *postgresRepository) GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]Order, error) {
	userOrdersQuery := `
		SELECT id, user_id, status, total_amount, shipping_address_text, created_at, updated_at
		FROM order_service.orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	orderRows, err := r.db.Query(ctx, userOrdersQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to query orders for user id %s: %w", userID, err)
	}
	defer orderRows.Close()

	ordersMap := make(map[uuid.UUID]*Order)
	var orderIDs []uuid.UUID

	for orderRows.Next() {
		var order Order
		err := orderRows.Scan(
			&order.ID,
			&order.UserID,
			&order.Status,
			&order.TotalAmount,
			&order.ShippingAddressText,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: failed scan order for user id %s: %w", userID, err)
		}
		order.OrderItems = make([]OrderItem, 0)
		ordersMap[order.ID] = &order
		orderIDs = append(orderIDs, order.ID)
	}

	if err = orderRows.Err(); err != nil {
		return nil, fmt.Errorf("repository: failed iterating orders for user id %s: %w", userID, err)
	}

	if len(orderIDs) == 0 {
		return []Order{}, nil
	}

	userOrderItemsQuery := `
		SELECT id, order_id, product_id, quantity, price_per_unit, created_at, updated_at
		FROM order_service.order_items
		WHERE order_id = ANY($1)
	`
	itemRows, err := r.db.Query(ctx, userOrderItemsQuery, orderIDs)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to query order items for user id %s: %w", userID, err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item OrderItem
		err := itemRows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Quantity,
			&item.PricePerUnit,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: failed to scan order item for user id %s: %w", userID, err)
		}

		if order, ok := ordersMap[item.OrderID]; ok {
			order.OrderItems = append(order.OrderItems, item)
		}
	}

	if err = itemRows.Err(); err != nil {
		return nil, fmt.Errorf("repository: failed iterating order itens by user id %s: %w", userID, err)
	}

	resultOrders := make([]Order, 0, len(ordersMap))

	for _, id := range orderIDs {
		if order, ok := ordersMap[id]; ok {
			resultOrders = append(resultOrders, *order)
		}
	}

	return resultOrders, nil
}
