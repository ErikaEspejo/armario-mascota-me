-- Migration: Create items table
-- Description: Products/inventory table linked to design assets

-- Table: items
-- Stores items with stock information linked to design assets
CREATE TABLE IF NOT EXISTS items (
    id BIGSERIAL PRIMARY KEY,
    design_asset_id BIGINT NOT NULL REFERENCES design_assets(id) ON DELETE RESTRICT,
    size TEXT NOT NULL,
    sku TEXT NOT NULL,
    price BIGINT NOT NULL CHECK (price >= 0),
    stock_total INT NOT NULL DEFAULT 0 CHECK (stock_total >= 0),
    stock_reserved INT NOT NULL DEFAULT 0 CHECK (stock_reserved >= 0),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(design_asset_id, size)
);

-- Indexes for items
CREATE INDEX IF NOT EXISTS idx_items_design_asset_id ON items(design_asset_id);
CREATE INDEX IF NOT EXISTS idx_items_size ON items(size);
CREATE INDEX IF NOT EXISTS idx_items_sku ON items(sku);
CREATE INDEX IF NOT EXISTS idx_items_is_active ON items(is_active);
CREATE INDEX IF NOT EXISTS idx_items_created_at ON items(created_at DESC);

-- Check constraint to ensure stock_reserved <= stock_total
ALTER TABLE items ADD CONSTRAINT check_stock_reserved_le_total 
    CHECK (stock_reserved <= stock_total);

