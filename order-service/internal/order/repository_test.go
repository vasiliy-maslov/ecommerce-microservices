package order_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/internal/order"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/pkg/config"
)

var db *sqlx.DB

func TestMain(m *testing.M) {
	// Задаём переменные окружения для тестов
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "123456")
	os.Setenv("DB_NAME", "orders")
	os.Setenv("DB_SSLMODE", "disable")
	os.Setenv("APP_PORT", "8080")

	cfg, err := config.Load("") // Пустой путь — используем переменные окружения
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.DBName, cfg.Postgres.SSLMode)
	log.Printf("Attempting to connect to database with: %s", connStr)

	db, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v (host=%s, port=%s, user=%s, dbname=%s, sslmode=%s)",
			err, cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.DBName, cfg.Postgres.SSLMode)
	}

	if db == nil {
		log.Fatalf("Database connection is nil after Connect")
	}

	// Проверяем подключение
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Запуск тестов
	os.Exit(m.Run())
}

func setup(t *testing.T) *order.PostgresOrderRepository {
	// Очистка таблицы перед тестом
	_, err := db.Exec("TRUNCATE TABLE orders RESTART IDENTITY")
	if err != nil {
		t.Fatalf("Failed to truncate table: %v", err)
	}

	// Создаём репозиторий
	repo := order.NewPostgresOrderRepository(db)

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

	ord := &order.Order{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		UserID:    "123e4567-e89b-12d3-a456-426614174000",
		Total:     100.50,
		Status:    "created",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx := context.Background()
	err := repo.Create(ctx, ord)
	assert.NoError(t, err, "Create should not return an error")

	// Проверяем, что заказ сохранился
	var savedOrder order.Order
	err = db.GetContext(ctx, &savedOrder, "SELECT * FROM orders WHERE id = $1", ord.ID)
	assert.NoError(t, err, "Should be able to retrieve the order")
	assert.Equal(t, ord.ID, savedOrder.ID, "Order ID should match")
	assert.Equal(t, ord.UserID, savedOrder.UserID, "UserID should match")
	assert.Equal(t, ord.Total, savedOrder.Total, "Total should match")
	assert.Equal(t, ord.Status, savedOrder.Status, "Status should match")
}

func TestPostgresOrderRepository_GetByID(t *testing.T) {
	repo := setup(t)

	// Сначала создаём заказ
	ord := &order.Order{
		ID:        "123e4567-e89b-12d3-a456-426614174001",
		UserID:    "123e4567-e89b-12d3-a456-426614174000",
		Total:     200.75,
		Status:    "paid",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ctx := context.Background()
	err := repo.Create(ctx, ord)
	assert.NoError(t, err, "Create should not return an error")

	// Теперь пытаемся получить заказ
	retrievedOrder, err := repo.GetByID(ctx, ord.ID)
	assert.NoError(t, err, "GetByID should not return an error")
	if assert.NotNil(t, retrievedOrder, "Retrieved order should not be nil") {
		assert.Equal(t, ord.ID, retrievedOrder.ID, "Order ID should match")
		assert.Equal(t, ord.UserID, retrievedOrder.UserID, "UserID should match")
		assert.Equal(t, ord.Total, retrievedOrder.Total, "Total should match")
		assert.Equal(t, ord.Status, retrievedOrder.Status, "Status should match")
	}
}
