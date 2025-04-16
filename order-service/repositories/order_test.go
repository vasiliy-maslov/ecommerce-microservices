package repositories_test

import (
	"context"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/entities"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/repositories"
	"log"
	"os"
	"testing"
	"time"
)

var db *sqlx.DB

func TestMain(m *testing.M) {
	var err error
	db, err = sqlx.Connect("postgres", "host=localhost port=5432 user=postgres password=123456 dbname=orders sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Запуск тестов
	os.Exit(m.Run())
}

func setup(t *testing.T) *repositories.PostgresOrderRepository {
	// Очистка таблицы перед тестом
	_, err := db.Exec("TRUNCATE TABLE orders RESTART IDENTITY")
	if err != nil {
		t.Fatalf("Failed to truncate table: %v", err)
	}

	// Создаём репозиторий
	repo := repositories.NewPostgresOrderRepository(db)

	// Очистка после теста
	t.Cleanup(func() {
		_, err := db.Exec("TRUNCATE TABLE orders RESTART IDENTITY")
		if err != nil {
			t.Fatalf("Failed to truncate table after test: %v", err)
		}
	})

	return repo
}

func TestPostgresOrderRepository_Create(t *testing.T) {
	repo := setup(t)

	order := &entities.Order{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		UserID:    "123e4567-e89b-12d3-a456-426614174000",
		Total:     100.50,
		Status:    "created",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx := context.Background()
	err := repo.Create(ctx, order)
	assert.NoError(t, err, "Create should not return an error")

	// Проверяем, что заказ сохранился
	var savedOrder entities.Order
	err = db.GetContext(ctx, &savedOrder, "SELECT * FROM orders WHERE id = $1", order.ID)
	assert.NoError(t, err, "Should be able to retrieve the order")
	assert.Equal(t, order.ID, savedOrder.ID, "Order ID should match")
	assert.Equal(t, order.UserID, savedOrder.UserID, "UserID should match")
	assert.Equal(t, order.Total, savedOrder.Total, "Total should match")
	assert.Equal(t, order.Status, savedOrder.Status, "Status should match")
}

func TestPostgresOrderRepository_GetByID(t *testing.T) {
	repo := setup(t)

	// Сначала создаём заказ
	order := &entities.Order{
		ID:        "123e4567-e89b-12d3-a456-426614174001",
		UserID:    "123e4567-e89b-12d3-a456-426614174000",
		Total:     200.75,
		Status:    "paid",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ctx := context.Background()
	err := repo.Create(ctx, order)
	assert.NoError(t, err, "Create should not return an error")

	// Теперь пытаемся получить заказ
	retrievedOrder, err := repo.GetByID(ctx, order.ID)
	assert.NoError(t, err, "GetByID should not return an error")
	if assert.NotNil(t, retrievedOrder, "Retrieved order should not be nil") {
		assert.Equal(t, order.ID, retrievedOrder.ID, "Order ID should match")
		assert.Equal(t, order.UserID, retrievedOrder.UserID, "UserID should match")
		assert.Equal(t, order.Total, retrievedOrder.Total, "Total should match")
		assert.Equal(t, order.Status, retrievedOrder.Status, "Status should match")
	}
}
