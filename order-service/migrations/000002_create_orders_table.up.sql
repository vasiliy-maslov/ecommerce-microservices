CREATE TABLE order_service.orders (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (
        status IN (
            'NEW',
            'PROCESSING',
            'PAID',
            'SHIPPED',
            'DELIVERED',
            'CANCELLED'
        )
    ),
    total_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    shipping_address_text TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX orders_user_id_idx ON order_service.orders (user_id);

CREATE INDEX orders_status_idx ON order_service.orders (status);