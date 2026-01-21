-- Migration: Add custom-ready status to design_assets
-- Description: Updates the CHECK constraint to include 'custom-ready' as a valid status value

-- Drop the existing CHECK constraint
ALTER TABLE design_assets DROP CONSTRAINT IF EXISTS design_assets_status_check;

-- Add the new CHECK constraint with 'custom-ready' included
ALTER TABLE design_assets ADD CONSTRAINT design_assets_status_check 
    CHECK (status IN ('pending', 'ready', 'custom-pending', 'custom-ready'));

