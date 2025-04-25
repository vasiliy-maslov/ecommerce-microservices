package order

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderRepository defines methods for interacting with orders in the database.
type OrderRepository interface {
	// Create inserts a new order into the database.
	Create(ctx context.Context, order *Order) error
	// GetByID retrieves an order by its ID.
	GetByID(ctx context.Context, id string) (*Order, error)
	// ExistsByID checks if an order with the given ID exists.
	ExistsByID(ctx context.Context, id string) (bool, error)
}

type PostgresOrderRepository struct {
	db *pgxpool.Pool
}

// NewPostgresOrderRepository creates a new PostgresOrderRepository.
func NewPostgresOrderRepository(db *pgxpool.Pool) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

// Create inserts a new order into the PostgreSQL database.
func (r *PostgresOrderRepository) Create(ctx context.Context, order *Order) error {
	query := `INSERT INTO orders (id, user_id, total, status, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query, order.ID, order.UserID, order.Total, order.Status, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error to create order: %w", err)
	}

	return nil
}

// GetByID retrieves an order by its ID from the PostgreSQL database.
func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (*Order, error) {
	var order Order
	query := `SELECT id, user_id, total, status, created_at, updated_at FROM orders WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&order.ID, &order.UserID, &order.Total, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error to get order by id: %v", err)
	}

	return &order, nil
}

// ExistsByID checks if an order with the given ID exists in the PostgreSQL database.
func (r *PostgresOrderRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS (SELECT 1 FROM orders WHERE id = $1)"
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check order existence: %w", err)
	}

	return exists, nil
}
