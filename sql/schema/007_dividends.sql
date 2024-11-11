-- sql/schema/007_dividends.sql

CREATE TABLE IF NOT EXISTS dividends (
    id TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    ex_date TIMESTAMPTZ,
    payment_date TIMESTAMPTZ,
    record_date TIMESTAMPTZ,
    declared_date TIMESTAMPTZ,
    amount NUMERIC,
    source TEXT,
    currency TEXT,
    frequency TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dividends_symbol_ex_date ON dividends(symbol, ex_date);
