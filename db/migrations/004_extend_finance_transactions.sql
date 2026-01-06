-- Migration: Extend finance_transactions table
-- Description: Add support for manual transactions by making source_id nullable,
-- adding default to source, adding counterparty field, and creating additional indexes

-- Make source_id nullable (for manual transactions)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'finance_transactions' AND column_name = 'source_id' AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE finance_transactions ALTER COLUMN source_id DROP NOT NULL;
    END IF;
END $$;

-- Add default 'manual' to source column
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'finance_transactions' AND column_name = 'source'
    ) THEN
        ALTER TABLE finance_transactions ALTER COLUMN source SET DEFAULT 'manual';
    END IF;
END $$;

-- Add counterparty column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'finance_transactions' AND column_name = 'counterparty'
    ) THEN
        ALTER TABLE finance_transactions ADD COLUMN counterparty TEXT NULL;
    END IF;
END $$;

-- Create index on (type, occurred_at DESC)
CREATE INDEX IF NOT EXISTS idx_finance_transactions_type_occurred_at 
ON finance_transactions(type, occurred_at DESC);

-- Create index on (destination, occurred_at DESC)
CREATE INDEX IF NOT EXISTS idx_finance_transactions_destination_occurred_at 
ON finance_transactions(destination, occurred_at DESC);

-- Create index on (category, occurred_at DESC)
CREATE INDEX IF NOT EXISTS idx_finance_transactions_category_occurred_at 
ON finance_transactions(category, occurred_at DESC);


