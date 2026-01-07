-- Migration: Create reserved_orders table
-- Description: Tables for managing reserved orders (persistent carts) with stock reservation

-- Table: reserved_orders
-- Stores reserved orders with status, assigned user, and customer information
CREATE TABLE IF NOT EXISTS reserved_orders (
    id BIGSERIAL PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('reserved', 'completed', 'canceled')),
    assigned_to TEXT NOT NULL,
    order_type TEXT NOT NULL CHECK (order_type IN ('detal', 'mayorista')),
    customer_name TEXT,
    customer_phone TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for reserved_orders
CREATE INDEX IF NOT EXISTS idx_reserved_orders_status ON reserved_orders(status);
CREATE INDEX IF NOT EXISTS idx_reserved_orders_created_at ON reserved_orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reserved_orders_order_type ON reserved_orders(order_type);

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

