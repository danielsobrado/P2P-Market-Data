-- sql/schema/002_peers.sql
CREATE TABLE IF NOT EXISTS peers (
    id TEXT PRIMARY KEY,
    address TEXT NOT NULL,
    reputation DECIMAL NOT NULL DEFAULT 0,
    last_seen TIMESTAMPTZ,
    roles TEXT[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_peers_reputation ON peers(reputation);
CREATE INDEX IF NOT EXISTS idx_peers_last_seen ON peers(last_seen);