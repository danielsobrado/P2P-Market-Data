-- sql/schema/005_transfers.sql
CREATE TABLE IF NOT EXISTS transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_peer_id TEXT NOT NULL REFERENCES peers(id),
    target_peer_id TEXT NOT NULL REFERENCES peers(id),
    data_type TEXT NOT NULL,
    symbol TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMPTZ,
    status TEXT NOT NULL,
    progress DECIMAL NOT NULL DEFAULT 0,
    size_bytes BIGINT NOT NULL,
    speed_bps DECIMAL,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_transfers_source ON transfers(source_peer_id);
CREATE INDEX IF NOT EXISTS idx_transfers_target ON transfers(target_peer_id);
CREATE INDEX IF NOT EXISTS idx_transfers_status ON transfers(status);
CREATE INDEX IF NOT EXISTS idx_transfers_start_time ON transfers(start_time);