-- Migration: Add order_type column to reserved_orders table
-- Description: Adds order_type field to existing reserved_orders table
-- This migration is safe to run even if the column already exists

-- Add order_type column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'reserved_orders' AND column_name = 'order_type'
    ) THEN
        ALTER TABLE reserved_orders ADD COLUMN order_type TEXT NOT NULL DEFAULT 'retail';
        -- Remove default after adding column (for future inserts)
        ALTER TABLE reserved_orders ALTER COLUMN order_type DROP DEFAULT;
    END IF;
END $$;

