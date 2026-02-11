-- sql/schema/008_insider_trades.sql
CREATE TABLE IF NOT EXISTS insider_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    trade_date TIMESTAMPTZ NOT NULL,
    insider_name TEXT NOT NULL,
    insider_title TEXT,
    transaction_type TEXT,
    shares BIGINT NOT NULL DEFAULT 0,
    price_per_share DECIMAL NOT NULL DEFAULT 0,
    value DECIMAL NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_insider_trades_symbol_date ON insider_trades(symbol, trade_date);
