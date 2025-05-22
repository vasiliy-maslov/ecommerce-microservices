package order_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/internal/order"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	// --- Получаем параметры БД для тестов ---
	// Пытаемся читать из ENV с суффиксом _TEST, иначе используем дефолты для localhost
	dbHost := os.Getenv("DB_HOST_TEST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT_TEST")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER_TEST")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD_TEST")
	if dbPassword == "" {
		dbPassword = "123456"
	}
	dbName := os.Getenv("DB_NAME_TEST")
	if dbName == "" {
		dbName = "ecommerce_db"
	}
	dbSSLMode := os.Getenv("DB_SSLMODE_TEST")
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}
	// --- КОНЕЦ Параметры БД ---

	// --- Установка соединения ---
	// Формируем строку подключения БЕЗ вызова config.NewConfig()
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=order_service",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// Используем стандартные настройки пула для тестов
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("host", dbHost).
			Str("port", dbPort).
			Str("user", dbUser).
			Str("dbname", dbName).
			Str("sslmode", dbSSLMode).
			Msg("Failed to connect to test database")
	}
	poolConfig.MaxConns = 5

	// Создаем контекст с таймаутом для подключения
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()

	testDB, err = pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		log.Fatal().Err(err).Str("db_host", dbHost).Str("db_port", dbPort).Msg("Failed to connect to test database")
	}

	// Пингуем с таймаутом
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err = testDB.Ping(pingCtx); err != nil {
		// Закрываем пул перед фатальной ошибкой, если он был создан
		if testDB != nil {
			testDB.Close()
		}
		log.Fatal().Err(err).Msg("Failed to ping test database")
	}
	log.Info().Msg("Test Database connection established.")
	// --- КОНЕЦ Установки соединения ---

	// Запуск тестов
	exitCode := m.Run()

	// Очистка
	if testDB != nil {
		testDB.Close()
		log.Info().Msg("TEST SETUP: Test Database connection closed.")
	}
	os.Exit(exitCode)
}

func truncateOrderTables(tb testing.TB, pool *pgxpool.Pool) {
	tb.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE TABLE order_service.order_items CASCADE")
	require.NoError(tb, err, "failed to truncate order_tems table")
	_, err = pool.Exec(context.Background(), "TRUNCATE TABLE order_service.orders CASCADE")
	require.NoError(tb, err, "failed to truncate orders table")
}
func TestOrderRepository_CreateOrder_Success(t *testing.T) {
	repo := order.NewRepository(testDB)
	ctx := context.Background()

	t.Cleanup(func() {
		truncateOrderTables(t, testDB)
	})

	// --- Arrange ---
	orderID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())
	productID1 := uuid.Must(uuid.NewV4())
	productID2 := uuid.Must(uuid.NewV4())

	orderToCreate := order.Order{
		ID:     orderID,
		UserID: userID,
		Status: order.StatusNew,
		OrderItems: []order.OrderItem{
			{
				ProductID:    productID1,
				Quantity:     2,
				PricePerUnit: 10.50,
			},
			{
				ProductID:    productID2,
				Quantity:     1,
				PricePerUnit: 25.00,
			},
		},
		TotalAmount:         46.00,
		ShippingAddressText: "123 Main St",
	}

	createdOrderID, err := repo.CreateOrder(ctx, &orderToCreate)

	require.NoError(t, err)
	require.Equal(t, orderToCreate.ID, createdOrderID)

	var fetchedOrder order.Order
	queryOrder := `SELECT id, user_id, status, total_amount, shipping_address_text, created_at, updated_at FROM order_service.orders WHERE id = $1`
	err = testDB.QueryRow(ctx, queryOrder, createdOrderID).Scan(
		&fetchedOrder.ID, &fetchedOrder.UserID, &fetchedOrder.Status,
		&fetchedOrder.TotalAmount, &fetchedOrder.ShippingAddressText,
		&fetchedOrder.CreatedAt, &fetchedOrder.UpdatedAt,
	)
	require.NoError(t, err)

	assert.Equal(t, orderToCreate.ID, fetchedOrder.ID)
	assert.Equal(t, orderToCreate.UserID, fetchedOrder.UserID)
	assert.Equal(t, orderToCreate.Status, fetchedOrder.Status)
	assert.InDelta(t, orderToCreate.TotalAmount, fetchedOrder.TotalAmount, 0.001)
	assert.Equal(t, orderToCreate.ShippingAddressText, fetchedOrder.ShippingAddressText)
	require.False(t, fetchedOrder.CreatedAt.IsZero())
	require.False(t, fetchedOrder.UpdatedAt.IsZero())

	queryOrderItems := `SELECT id, order_id, product_id, quantity, price_per_unit, created_at, updated_at
	                    FROM order_service.order_items WHERE order_id = $1 ORDER BY product_id`

	rows, err := testDB.Query(ctx, queryOrderItems, createdOrderID)
	require.NoError(t, err)
	defer rows.Close()

	var fetchedItems []order.OrderItem
	for rows.Next() {
		var item order.OrderItem
		err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
			&item.PricePerUnit, &item.CreatedAt, &item.UpdatedAt,
		)
		require.NoError(t, err)
		fetchedItems = append(fetchedItems, item)
	}
	require.NoError(t, rows.Err())
	require.Len(t, fetchedItems, len(orderToCreate.OrderItems))

	sort.Slice(orderToCreate.OrderItems, func(i, j int) bool {
		return orderToCreate.OrderItems[i].ProductID.String() < orderToCreate.OrderItems[j].ProductID.String()
	})

	for i, expectedItem := range orderToCreate.OrderItems {
		fetchedItem := fetchedItems[i]
		assert.Equal(t, createdOrderID, fetchedItem.OrderID)
		assert.Equal(t, expectedItem.ProductID, fetchedItem.ProductID)
		assert.Equal(t, expectedItem.Quantity, fetchedItem.Quantity)
		assert.InDelta(t, expectedItem.PricePerUnit, fetchedItem.PricePerUnit, 0.001)
		require.False(t, fetchedItem.CreatedAt.IsZero())
		require.False(t, fetchedItem.UpdatedAt.IsZero())
		require.NotEqual(t, uuid.Nil, fetchedItem.ID)
	}
}

func TestOrderRepository_GetOrderByID_Success(t *testing.T) {
	repo := order.NewRepository(testDB)
	ctx := context.Background()

	t.Cleanup(func() {
		truncateOrderTables(t, testDB)
	})

	// --- Arrange ---
	orderID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())
	productID1 := uuid.Must(uuid.NewV4())
	productID2 := uuid.Must(uuid.NewV4())

	// Определяем ожидаемый заказ с его позициями
	expectedOrder := order.Order{
		ID:     orderID,
		UserID: userID,
		Status: order.StatusNew,
		OrderItems: []order.OrderItem{
			{
				ID:           uuid.Must(uuid.NewV4()), // Генерируем ID для позиции
				OrderID:      orderID,                 // Связываем с заказом
				ProductID:    productID1,
				Quantity:     2,
				PricePerUnit: 10.50,
			},
			{
				ID:           uuid.Must(uuid.NewV4()), // Генерируем ID для позиции
				OrderID:      orderID,                 // Связываем с заказом
				ProductID:    productID2,
				Quantity:     1,
				PricePerUnit: 25.00,
			},
		},
		TotalAmount:         46.00,
		ShippingAddressText: "123 Main St",
	}

	// Вставляем основной заказ в БД
	orderCreatedAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Microsecond)
	orderUpdatedAt := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Microsecond)

	queryOrder := `
		INSERT INTO order_service.orders (id, user_id, status, total_amount, shipping_address_text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := testDB.Exec(ctx, queryOrder,
		expectedOrder.ID,
		expectedOrder.UserID,
		string(expectedOrder.Status),
		expectedOrder.TotalAmount,
		expectedOrder.ShippingAddressText,
		orderCreatedAt,
		orderUpdatedAt,
	)
	require.NoError(t, err)
	expectedOrder.CreatedAt = orderCreatedAt
	expectedOrder.UpdatedAt = orderUpdatedAt

	// Вставляем позиции заказа в БД
	queryOrderItems := `
		INSERT INTO order_service.order_items (id, order_id, product_id, quantity, price_per_unit, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	for i := range expectedOrder.OrderItems { // Используем индекс для модификации оригинала
		// Генерируем уникальные CreatedAt/UpdatedAt для каждой позиции для большей точности теста
		itemCreatedAt := time.Now().Add(-2 * time.Hour).Add(time.Duration(i) * time.Minute).UTC().Truncate(time.Microsecond)
		itemUpdatedAt := time.Now().Add(-1 * time.Hour).Add(time.Duration(i) * time.Minute).UTC().Truncate(time.Microsecond)

		_, err = testDB.Exec(ctx, queryOrderItems,
			expectedOrder.OrderItems[i].ID, // Используем ID, заданный в expectedOrder.OrderItems
			expectedOrder.ID,               // Это order_id для связи
			expectedOrder.OrderItems[i].ProductID,
			expectedOrder.OrderItems[i].Quantity,
			expectedOrder.OrderItems[i].PricePerUnit,
			itemCreatedAt,
			itemUpdatedAt,
		)
		require.NoError(t, err)
		expectedOrder.OrderItems[i].CreatedAt = itemCreatedAt // Сохраняем для сравнения
		expectedOrder.OrderItems[i].UpdatedAt = itemUpdatedAt // Сохраняем для сравнения
	}

	// --- Act ---
	returnedOrder, err := repo.GetOrderByID(ctx, expectedOrder.ID)

	// --- Assert ---
	require.NoError(t, err)
	require.NotNil(t, returnedOrder)

	// Сравниваем поля основного заказа
	assert.Equal(t, expectedOrder.ID, returnedOrder.ID)
	assert.Equal(t, expectedOrder.UserID, returnedOrder.UserID)
	assert.Equal(t, expectedOrder.Status, returnedOrder.Status)
	assert.InDelta(t, expectedOrder.TotalAmount, returnedOrder.TotalAmount, 0.001)
	assert.Equal(t, expectedOrder.ShippingAddressText, returnedOrder.ShippingAddressText)
	assert.True(t, expectedOrder.CreatedAt.Equal(returnedOrder.CreatedAt), "Order CreatedAt mismatch: expected %v, got %v", expectedOrder.CreatedAt, returnedOrder.CreatedAt)
	assert.True(t, expectedOrder.UpdatedAt.Equal(returnedOrder.UpdatedAt), "Order UpdatedAt mismatch: expected %v, got %v", expectedOrder.UpdatedAt, returnedOrder.UpdatedAt)

	// Сравниваем позиции заказа
	// Сортируем оба слайса перед сравнением по ID элемента заказа для детерминизма
	sort.Slice(expectedOrder.OrderItems, func(i, j int) bool {
		return expectedOrder.OrderItems[i].ID.String() < expectedOrder.OrderItems[j].ID.String()
	})
	sort.Slice(returnedOrder.OrderItems, func(i, j int) bool {
		return returnedOrder.OrderItems[i].ID.String() < returnedOrder.OrderItems[j].ID.String()
	})

	require.Len(t, returnedOrder.OrderItems, len(expectedOrder.OrderItems), "Number of order items mismatch")

	for i, returnedItem := range returnedOrder.OrderItems {
		expectedItem := expectedOrder.OrderItems[i]

		assert.Equal(t, expectedItem.ID, returnedItem.ID, "OrderItem ID mismatch for item %d", i)
		assert.Equal(t, expectedOrder.ID, returnedItem.OrderID, "OrderItem OrderID mismatch for item %d (should be parent order ID)", i) // Проверяем связь с родителем
		assert.Equal(t, expectedItem.ProductID, returnedItem.ProductID, "OrderItem ProductID mismatch for item %d", i)
		assert.Equal(t, expectedItem.Quantity, returnedItem.Quantity, "OrderItem Quantity mismatch for item %d", i)
		assert.InDelta(t, expectedItem.PricePerUnit, returnedItem.PricePerUnit, 0.001, "OrderItem PricePerUnit mismatch for item %d", i)
		assert.True(t, expectedItem.CreatedAt.Equal(returnedItem.CreatedAt), "OrderItem CreatedAt mismatch for item %d: expected %v, got %v", i, expectedItem.CreatedAt, returnedItem.CreatedAt)
		assert.True(t, expectedItem.UpdatedAt.Equal(returnedItem.UpdatedAt), "OrderItem UpdatedAt mismatch for item %d: expected %v, got %v", i, expectedItem.UpdatedAt, returnedItem.UpdatedAt)
	}
}

func TestOrderRepository_GetOrderByID_NotFound(t *testing.T) {
	repo := order.NewRepository(testDB)

	nonExistentID := uuid.Must(uuid.NewV4())

	fetchedOrder, err := repo.GetOrderByID(context.Background(), nonExistentID)
	require.Error(t, err)
	require.ErrorIs(t, err, order.ErrOrderNotFound)
	require.Nil(t, fetchedOrder)
}

func TestOrderRepository_UpdateOrderStatus_Success(t *testing.T) {
	repo := order.NewRepository(testDB)
	ctx := context.Background()

	t.Cleanup(func() {
		truncateOrderTables(t, testDB)
	})

	orderID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())

	initialOrder := order.Order{
		ID:                  orderID,
		UserID:              userID,
		Status:              order.StatusNew,
		TotalAmount:         46.00,
		ShippingAddressText: "123 Main St",
	}

	orderUpdatedAt := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Microsecond)

	queryInsertOrder := `
		INSERT INTO order_service.orders (id, user_id, status, total_amount, shipping_address_text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := testDB.Exec(ctx, queryInsertOrder,
		initialOrder.ID,
		initialOrder.UserID,
		string(initialOrder.Status),
		initialOrder.TotalAmount,
		initialOrder.ShippingAddressText,
		time.Now(),
		orderUpdatedAt,
	)
	require.NoError(t, err)
	initialOrder.UpdatedAt = orderUpdatedAt

	newStatus := order.StatusProcessing

	err = repo.UpdateOrderStatus(ctx, orderID, newStatus)
	require.NoError(t, err)

	var fetchedStatusStr string
	var fetchedUpdatedAt time.Time

	querySelect := `SELECT status, updated_at FROM order_service.orders WHERE id = $1`
	err = testDB.QueryRow(ctx, querySelect, initialOrder.ID).Scan(&fetchedStatusStr, &fetchedUpdatedAt)
	require.NoError(t, err, "Failed to query updated order from DB")

	assert.Equal(t, string(newStatus), fetchedStatusStr, "Order status was not updated correctly")
	assert.True(t, fetchedUpdatedAt.After(initialOrder.UpdatedAt), "UpdatedAt should be after initial UpdatedAt. Initial: %v, Fetched: %v", initialOrder.UpdatedAt, fetchedUpdatedAt)
}

func TestOrderRepository_UpdateOrderStatus_NotFound(t *testing.T) {
	repo := order.NewRepository(testDB)

	nonExistentID := uuid.Must(uuid.NewV4())
	newStatus := order.StatusNew

	err := repo.UpdateOrderStatus(context.Background(), nonExistentID, newStatus)
	require.Error(t, err)
	require.ErrorIs(t, err, order.ErrOrderNotFound)
}

func TestOrderRepository_GetOrdersByUserID_Success_WithOrders(t *testing.T) {
	repo := order.NewRepository(testDB)
	ctx := context.Background()

	t.Cleanup(func() {
		truncateOrderTables(t, testDB)
	})

	userID := uuid.Must(uuid.NewV4())
	orderID1 := uuid.Must(uuid.NewV4())
	orderID2 := uuid.Must(uuid.NewV4())

	now := time.Now().UTC()

	expectedOrders := []order.Order{
		{
			ID:                  orderID1,
			UserID:              userID,
			Status:              order.StatusNew,
			TotalAmount:         55.00,
			ShippingAddressText: "Address 1",
			OrderItems: []order.OrderItem{
				{
					ID:           uuid.Must(uuid.NewV4()),
					OrderID:      orderID1,
					ProductID:    uuid.Must(uuid.NewV4()),
					Quantity:     1,
					PricePerUnit: 25.00,
					CreatedAt:    now.Add(-4 * time.Hour).Truncate(time.Microsecond),
					UpdatedAt:    now.Add(-3 * time.Hour).Truncate(time.Microsecond),
				},
				{
					ID:           uuid.Must(uuid.NewV4()),
					OrderID:      orderID1,
					ProductID:    uuid.Must(uuid.NewV4()),
					Quantity:     3,
					PricePerUnit: 10.00,
					CreatedAt:    now.Add(-4 * time.Hour).Truncate(time.Microsecond),
					UpdatedAt:    now.Add(-3 * time.Hour).Truncate(time.Microsecond),
				},
			},
			CreatedAt: now.Add(-4 * time.Hour).Truncate(time.Microsecond),
			UpdatedAt: now.Add(-3 * time.Hour).Truncate(time.Microsecond),
		},
		{
			ID:                  orderID2,
			UserID:              userID,
			Status:              order.StatusProcessing,
			TotalAmount:         65.00,
			ShippingAddressText: "Address 2",
			OrderItems: []order.OrderItem{
				{
					ID:           uuid.Must(uuid.NewV4()),
					OrderID:      orderID2,
					ProductID:    uuid.Must(uuid.NewV4()),
					Quantity:     5,
					PricePerUnit: 5.00,
					CreatedAt:    now.Add(-2 * time.Hour).Truncate(time.Microsecond),
					UpdatedAt:    now.Add(-1 * time.Hour).Truncate(time.Microsecond),
				},
				{
					ID:           uuid.Must(uuid.NewV4()),
					OrderID:      orderID2,
					ProductID:    uuid.Must(uuid.NewV4()),
					Quantity:     2,
					PricePerUnit: 20.00,
					CreatedAt:    now.Add(-2 * time.Hour).Truncate(time.Microsecond),
					UpdatedAt:    now.Add(-1 * time.Hour).Truncate(time.Microsecond),
				},
			},
			CreatedAt: now.Add(-2 * time.Hour).Truncate(time.Microsecond),
			UpdatedAt: now.Add(-1 * time.Hour).Truncate(time.Microsecond),
		},
	}

	queryInsertOrders := `
		INSERT INTO order_service.orders (id, user_id, status, total_amount, shipping_address_text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	queryInsertOrderItems := `
		INSERT INTO order_service.order_items (id, order_id, product_id, quantity, price_per_unit, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	for _, orderToInsert := range expectedOrders {
		_, err := testDB.Exec(ctx, queryInsertOrders,
			orderToInsert.ID, orderToInsert.UserID, string(orderToInsert.Status),
			orderToInsert.TotalAmount, orderToInsert.ShippingAddressText,
			orderToInsert.CreatedAt, orderToInsert.UpdatedAt,
		)
		require.NoError(t, err)

		for _, itemToInsert := range orderToInsert.OrderItems {
			_, err := testDB.Exec(ctx, queryInsertOrderItems,
				itemToInsert.ID, itemToInsert.OrderID, itemToInsert.ProductID,
				itemToInsert.Quantity, itemToInsert.PricePerUnit,
				itemToInsert.CreatedAt, itemToInsert.UpdatedAt,
			)
			require.NoError(t, err)
		}
	}

	returnedOrders, err := repo.GetOrdersByUserID(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, returnedOrders)
	require.Len(t, returnedOrders, len(expectedOrders), "Number of orders mismatch")

	sort.Slice(expectedOrders, func(i, j int) bool {
		return expectedOrders[i].CreatedAt.After(expectedOrders[j].CreatedAt) // DESC order
	})

	for i, expected := range expectedOrders {
		returned := returnedOrders[i]

		assert.Equal(t, expected.ID, returned.ID, "Order ID mismatch for order at index %d", i)
		assert.Equal(t, expected.UserID, returned.UserID, "UserID mismatch for order %s", expected.ID)
		assert.Equal(t, expected.Status, returned.Status, "Status mismatch for order %s", expected.ID)
		assert.InDelta(t, expected.TotalAmount, returned.TotalAmount, 0.001, "TotalAmount mismatch for order %s", expected.ID)
		assert.Equal(t, expected.ShippingAddressText, returned.ShippingAddressText, "ShippingAddressText mismatch for order %s", expected.ID)
		assert.True(t, expected.CreatedAt.Equal(returned.CreatedAt), "CreatedAt mismatch for order %s. Expected: %v, Got: %v", expected.ID, expected.CreatedAt, returned.CreatedAt)
		assert.True(t, expected.UpdatedAt.Equal(returned.UpdatedAt), "UpdatedAt mismatch for order %s. Expected: %v, Got: %v", expected.ID, expected.UpdatedAt, returned.UpdatedAt)

		sort.Slice(expected.OrderItems, func(k, l int) bool {
			return expected.OrderItems[k].ID.String() < expected.OrderItems[l].ID.String()
		})
		sort.Slice(returned.OrderItems, func(k, l int) bool {
			return returned.OrderItems[k].ID.String() < returned.OrderItems[l].ID.String()
		})

		require.Len(t, returned.OrderItems, len(expected.OrderItems), "Number of order items mismatch for order %s", expected.ID)

		for j, expectedItem := range expected.OrderItems {
			returnedItem := returned.OrderItems[j]

			assert.Equal(t, expectedItem.ID, returnedItem.ID, "OrderItem ID mismatch for order %s, item at index %d", expected.ID, j)
			assert.Equal(t, expected.ID, returnedItem.OrderID, "OrderItem OrderID mismatch for order %s, item %s (should be parent order ID)", expected.ID, expectedItem.ID)
			assert.Equal(t, expectedItem.ProductID, returnedItem.ProductID, "OrderItem ProductID mismatch for order %s, item %s", expected.ID, expectedItem.ID)
			assert.Equal(t, expectedItem.Quantity, returnedItem.Quantity, "OrderItem Quantity mismatch for order %s, item %s", expected.ID, expectedItem.ID)
			assert.InDelta(t, expectedItem.PricePerUnit, returnedItem.PricePerUnit, 0.001, "OrderItem PricePerUnit mismatch for order %s, item %s", expected.ID, expectedItem.ID)
			assert.True(t, expectedItem.CreatedAt.Equal(returnedItem.CreatedAt), "OrderItem CreatedAt mismatch for order %s, item %s. Expected: %v, Got: %v", expected.ID, expectedItem.ID, expectedItem.CreatedAt, returnedItem.CreatedAt)
			assert.True(t, expectedItem.UpdatedAt.Equal(returnedItem.UpdatedAt), "OrderItem UpdatedAt mismatch for order %s, item %s. Expected: %v, Got: %v", expected.ID, expectedItem.ID, expectedItem.UpdatedAt, returnedItem.UpdatedAt)
		}
	}
}

func TestOrderRepository_GetOrdersByUserID_Success_NoOrders(t *testing.T) {
	repo := order.NewRepository(testDB)
	ctx := context.Background()

	t.Cleanup(func() {
		truncateOrderTables(t, testDB)
	})

	userIDWithoutOrders := uuid.Must(uuid.NewV4())

	returnedOrders, err := repo.GetOrdersByUserID(ctx, userIDWithoutOrders)
	require.NoError(t, err)
	require.NotNil(t, returnedOrders)
	require.Len(t, returnedOrders, 0)
}
