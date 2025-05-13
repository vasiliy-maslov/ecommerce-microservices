CREATE TABLE order_service.order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES order_service.orders (id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price_per_unit DECIMAL(10, 2) NOT NULL CHECK (price_per_unit >= 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX order_items_order_id_idx ON order_service.order_items (order_id);

CREATE INDEX order_items_product_id_idx ON order_service.order_items (product_id);