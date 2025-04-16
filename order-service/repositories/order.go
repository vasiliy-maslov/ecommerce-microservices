package repositories

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/entities"
)

type OrderRepository interface {
	Create(ctx context.Context, order *entities.Order) error
	GetByID(ctx context.Context, id string) (*entities.Order, error)
}

type PostgresOrderRepository struct {
	db *sqlx.DB
}

func NewPostgresOrderRepository(db *sqlx.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order *entities.Order) error {
	query := `INSERT INTO orders (id, user_id, total, status, created_at, updated_at)
              VALUES (:id, :user_id, :total, :status, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, order)
	if err != nil {
		return fmt.Errorf("Error to create order: %v", err)
	}

	return nil
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (*entities.Order, error) {
	var order entities.Order
	query := `SELECT * FROM orders WHERE id = $1`
	err := r.db.GetContext(ctx, &order, query)
	if err != nil {
		return nil, fmt.Errorf("Error to get order by id: %v", err)
	}

	return &order, nil
}
