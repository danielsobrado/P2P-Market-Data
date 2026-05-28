-- sql/schema/010_splits.sql
CREATE TABLE IF NOT EXISTS splits (
    id TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    split_ratio DECIMAL NOT NULL,
    announcement_date TIMESTAMPTZ,
    ex_date TIMESTAMPTZ NOT NULL,
    old_shares INTEGER NOT NULL,
    new_shares INTEGER NOT NULL,
    status TEXT NOT NULL,
    source TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_splits_symbol_ex_date ON splits(symbol, ex_date);
