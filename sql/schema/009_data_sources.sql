-- sql/schema/009_data_sources.sql
CREATE TABLE IF NOT EXISTS data_sources (
    peer_id TEXT PRIMARY KEY,
    reputation DECIMAL NOT NULL DEFAULT 1,
    data_types TEXT[] NOT NULL DEFAULT '{}',
    available_symbols TEXT[] NOT NULL DEFAULT '{}',
    data_range_start TIMESTAMPTZ,
    data_range_end TIMESTAMPTZ,
    last_update TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reliability DECIMAL NOT NULL DEFAULT 1,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_data_sources_symbols ON data_sources USING GIN(available_symbols);
CREATE INDEX IF NOT EXISTS idx_data_sources_types ON data_sources USING GIN(data_types);
