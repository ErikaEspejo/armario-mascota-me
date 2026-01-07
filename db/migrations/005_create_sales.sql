-- Migration: Create sales table
-- Description: Table for managing sales records linked to reserved orders

-- Table: sales
-- Stores sales records linked to reserved orders
CREATE TABLE IF NOT EXISTS sales (
    id BIGSERIAL PRIMARY KEY,
    reserved_order_id BIGINT NOT NULL UNIQUE REFERENCES reserved_orders(id) ON DELETE RESTRICT,
    sold_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    customer_name TEXT,
    amount_paid BIGINT NOT NULL CHECK (amount_paid > 0),
    payment_method TEXT NOT NULL CHECK (payment_method != ''),
    payment_destination TEXT NOT NULL CHECK (payment_destination != ''),
    status TEXT NOT NULL DEFAULT 'paid' CHECK (status IN ('paid', 'refunded', 'pending')),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for sales
CREATE INDEX IF NOT EXISTS idx_sales_sold_at ON sales(sold_at DESC);
CREATE INDEX IF NOT EXISTS idx_sales_reserved_order_id ON sales(reserved_order_id);
CREATE INDEX IF NOT EXISTS idx_sales_status ON sales(status);
CREATE INDEX IF NOT EXISTS idx_sales_created_at ON sales(created_at DESC);

