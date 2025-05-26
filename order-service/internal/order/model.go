// order-service/internal/order/model.go
package order

import (
	"github.com/gofrs/uuid"
	"time"
)

type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusPaid       OrderStatus = "PAID"
	StatusShipped    OrderStatus = "SHIPPED"
	StatusDelivered  OrderStatus = "DELIVERED"
	StatusCancelled  OrderStatus = "CANCELLED"
)

func (os OrderStatus) String() string {
	return string(os)
}

type OrderItem struct {
	ID           uuid.UUID `json:"id" db:"id"`
	OrderID      uuid.UUID `json:"order_id" db:"order_id"`
	ProductID    uuid.UUID `json:"product_id" db:"product_id"`
	Quantity     int       `json:"quantity" db:"quantity"`
	PricePerUnit float64   `json:"price_per_unit" db:"price_per_unit"` // Используем float64 для денег, или специальный тип decimal
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Order struct {
	ID                  uuid.UUID   `json:"id" db:"id"`
	UserID              uuid.UUID   `json:"user_id" db:"user_id"`
	Status              OrderStatus `json:"status" db:"status"`
	OrderItems          []OrderItem `json:"order_items" db:"-"` // Не хранится напрямую в таблице orders, а получается JOIN'ом
	TotalAmount         float64     `json:"total_amount" db:"total_amount"`
	ShippingAddressText string      `json:"shipping_address_text,omitempty" db:"shipping_address_text"`
	CreatedAt           time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at" db:"updated_at"`
}
