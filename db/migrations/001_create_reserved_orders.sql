-- Migration: Create reserved_orders and reserved_order_lines tables
-- Description: Tables for managing reserved orders (persistent carts) with stock reservation

-- Table: reserved_orders
-- Stores reserved orders with status, assigned user, and customer information
CREATE TABLE IF NOT EXISTS reserved_orders (
    id BIGSERIAL PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('reserved', 'completed', 'canceled')),
    assigned_to TEXT NOT NULL,
    order_type TEXT NOT NULL,
    customer_name TEXT,
    customer_phone TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for reserved_orders
CREATE INDEX IF NOT EXISTS idx_reserved_orders_status ON reserved_orders(status);
CREATE INDEX IF NOT EXISTS idx_reserved_orders_created_at ON reserved_orders(created_at);

-- Table: reserved_order_lines
-- Stores line items for reserved orders with quantity and unit price
CREATE TABLE IF NOT EXISTS reserved_order_lines (
    id BIGSERIAL PRIMARY KEY,
    reserved_order_id BIGINT NOT NULL REFERENCES reserved_orders(id) ON DELETE CASCADE,
    item_id BIGINT NOT NULL REFERENCES items(id),
    qty INT NOT NULL CHECK (qty > 0),
    unit_price BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(reserved_order_id, item_id)
);

-- Indexes for reserved_order_lines
CREATE INDEX IF NOT EXISTS idx_reserved_order_lines_order_id ON reserved_order_lines(reserved_order_id);
CREATE INDEX IF NOT EXISTS idx_reserved_order_lines_item_id ON reserved_order_lines(item_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_reserved_orders_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at
CREATE TRIGGER trigger_update_reserved_orders_updated_at
    BEFORE UPDATE ON reserved_orders
    FOR EACH ROW
    EXECUTE FUNCTION update_reserved_orders_updated_at();

