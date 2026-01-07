-- Migration: Create finance_transactions table
-- Description: Table for managing financial transactions (ledger entries)

-- Table: finance_transactions
-- Stores financial transactions (ledger entries)
CREATE TABLE IF NOT EXISTS finance_transactions (
    id BIGSERIAL PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('income', 'expense')),
    source TEXT NOT NULL DEFAULT 'manual',
    source_id BIGINT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    amount BIGINT NOT NULL CHECK (amount > 0),
    destination TEXT NOT NULL CHECK (destination != ''),
    category TEXT,
    counterparty TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for finance_transactions
CREATE INDEX IF NOT EXISTS idx_finance_transactions_occurred_at ON finance_transactions(occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_finance_transactions_source_source_id ON finance_transactions(source, source_id);
CREATE INDEX IF NOT EXISTS idx_finance_transactions_type_occurred_at ON finance_transactions(type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_finance_transactions_destination_occurred_at ON finance_transactions(destination, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_finance_transactions_category_occurred_at ON finance_transactions(category, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_finance_transactions_type ON finance_transactions(type);
CREATE INDEX IF NOT EXISTS idx_finance_transactions_created_at ON finance_transactions(created_at DESC);

