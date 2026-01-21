-- Migration: Add custom_code column to reserved_order_lines table
-- Description: Adds nullable custom_code field to store custom item codes when type=custom

-- Add custom_code column to reserved_order_lines table
ALTER TABLE reserved_order_lines
ADD COLUMN IF NOT EXISTS custom_code TEXT;

