-- Migration: Create design_assets table
-- Description: Base table for design assets from Google Drive

-- Table: design_assets
-- Stores design assets with metadata and status
CREATE TABLE IF NOT EXISTS design_assets (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL,
    description TEXT,
    drive_file_id TEXT NOT NULL UNIQUE,
    image_url TEXT NOT NULL,
    color_primary TEXT,
    color_secondary TEXT,
    hoodie_type TEXT,
    image_type TEXT,
    deco_id TEXT,
    deco_base TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    has_highlights BOOLEAN NOT NULL DEFAULT false,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ready')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for design_assets
CREATE INDEX IF NOT EXISTS idx_design_assets_status ON design_assets(status);
CREATE INDEX IF NOT EXISTS idx_design_assets_is_active ON design_assets(is_active);
CREATE INDEX IF NOT EXISTS idx_design_assets_drive_file_id ON design_assets(drive_file_id);
CREATE INDEX IF NOT EXISTS idx_design_assets_created_at ON design_assets(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_design_assets_code ON design_assets(code);

