-- Migration: Create sales and finance_transactions tables
-- Description: Tables for managing sales and financial transactions

-- Table: sales
-- Stores sales records linked to reserved orders
CREATE TABLE IF NOT EXISTS sales (
    id BIGSERIAL PRIMARY KEY,
    reserved_order_id BIGINT NOT NULL UNIQUE REFERENCES reserved_orders(id),
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

-- Table: finance_transactions
-- Stores financial transactions (ledger entries)
CREATE TABLE IF NOT EXISTS finance_transactions (
    id BIGSERIAL PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('income', 'expense')),
    source TEXT NOT NULL,
    source_id BIGINT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    amount BIGINT NOT NULL CHECK (amount > 0),
    destination TEXT NOT NULL CHECK (destination != ''),
    category TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for finance_transactions
CREATE INDEX IF NOT EXISTS idx_finance_transactions_occurred_at ON finance_transactions(occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_finance_transactions_source_source_id ON finance_transactions(source, source_id);


