package order

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
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
	db *sqlx.DB
}

// NewPostgresOrderRepository creates a new PostgresOrderRepository.
func NewPostgresOrderRepository(db *sqlx.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

// Create inserts a new order into the PostgreSQL database.
func (r *PostgresOrderRepository) Create(ctx context.Context, order *Order) error {
	query := `INSERT INTO orders (id, user_id, total, status, created_at, updated_at)
              VALUES (:id, :user_id, :total, :status, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, order)
	if err != nil {
		return fmt.Errorf("error to create order: %w", err)
	}

	return nil
}

// GetByID retrieves an order by its ID from the PostgreSQL database.
func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (*Order, error) {
	var order Order
	query := `SELECT * FROM orders WHERE id = $1`
	err := r.db.GetContext(ctx, &order, query, id)
	if err == sql.ErrNoRows {
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
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check order existence: %w", err)
	}

	return exists, nil
}
