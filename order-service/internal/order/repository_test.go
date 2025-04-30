package order_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/internal/config"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/internal/order"
)

var db *pgxpool.Pool

func TestMain(m *testing.M) {
	// Задаём переменные окружения для тестов
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "123456")
	os.Setenv("DB_NAME", "orders")
	os.Setenv("DB_SSLMODE", "disable")
	os.Setenv("APP_PORT", "8080")

	cfg := config.Config{
		Postgres: config.PostgresConfig{
			Host:            os.Getenv("DB_HOST"),
			Port:            os.Getenv("DB_PORT"),
			User:            os.Getenv("DB_USER"),
			Password:        os.Getenv("DB_PASSWORD"),
			DBName:          os.Getenv("DB_NAME"),
			SSLMode:         os.Getenv("DB_SSLMODE"),
			MaxConns:        10,
			MinConns:        2,
			MaxConnLifetime: 30 * time.Minute,
		},
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.DBName, cfg.Postgres.SSLMode)
	log.Printf("Attempting to connect to database with: %s", connStr)

	var err error
	db, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v (host=%s, port=%s, user=%s, dbname=%s, sslmode=%s)",
			err, cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.DBName, cfg.Postgres.SSLMode)
	}

	exitCode := m.Run()

	db.Close()

	os.Exit(exitCode)
}

func setup(t *testing.T) *order.PostgresOrderRepository {
	// Очистка таблицы перед тестом
	_, err := db.Exec(context.Background(), "TRUNCATE TABLE orders RESTART IDENTITY")
	if err != nil {
		t.Fatalf("Failed to truncate table: %v", err)
	}

	// Создаём репозиторий
	repo := order.NewPostgresOrderRepository(db)

	// Очистка после теста
	t.Cleanup(func() {
		_, err := db.Exec(context.Background(), "TRUNCATE TABLE orders RESTART IDENTITY")
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
	err = db.QueryRow(context.Background(), "SELECT id, user_id, total, status, created_at, updated_at FROM orders WHERE id = $1", ord.ID).
		Scan(&savedOrder.ID, &savedOrder.UserID, &savedOrder.Total, &savedOrder.Status, &savedOrder.CreatedAt, &savedOrder.UpdatedAt)
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
