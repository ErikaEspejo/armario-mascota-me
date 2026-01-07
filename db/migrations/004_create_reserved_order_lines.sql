-- Migration: Create reserved_order_lines table
-- Description: Line items for reserved orders with quantity and unit price

-- Table: reserved_order_lines
-- Stores line items for reserved orders with quantity and unit price
CREATE TABLE IF NOT EXISTS reserved_order_lines (
    id BIGSERIAL PRIMARY KEY,
    reserved_order_id BIGINT NOT NULL REFERENCES reserved_orders(id) ON DELETE CASCADE,
    item_id BIGINT NOT NULL REFERENCES items(id) ON DELETE RESTRICT,
    qty INT NOT NULL CHECK (qty > 0),
    unit_price BIGINT NOT NULL CHECK (unit_price >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(reserved_order_id, item_id)
);

-- Indexes for reserved_order_lines
CREATE INDEX IF NOT EXISTS idx_reserved_order_lines_order_id ON reserved_order_lines(reserved_order_id);
CREATE INDEX IF NOT EXISTS idx_reserved_order_lines_item_id ON reserved_order_lines(item_id);
CREATE INDEX IF NOT EXISTS idx_reserved_order_lines_created_at ON reserved_order_lines(created_at DESC);

