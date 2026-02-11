-- sql/schema/003_votes.sql
CREATE TABLE IF NOT EXISTS votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_data_id UUID NOT NULL REFERENCES market_data(id),
    peer_id TEXT REFERENCES peers(id),
    validator_id TEXT,
    vote_type TEXT,
    is_valid BOOLEAN,
    confidence DECIMAL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    signature BYTEA,
    reason TEXT,
    metadata JSONB,
    UNIQUE(market_data_id, validator_id)
);

CREATE INDEX IF NOT EXISTS idx_votes_market_data ON votes(market_data_id);
CREATE INDEX IF NOT EXISTS idx_votes_peer ON votes(peer_id);
CREATE INDEX IF NOT EXISTS idx_votes_timestamp ON votes(timestamp);
