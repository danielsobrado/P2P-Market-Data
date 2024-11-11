-- sql/schema/003_votes.sql
CREATE TABLE IF NOT EXISTS votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_data_id UUID NOT NULL REFERENCES market_data(id),
    peer_id TEXT NOT NULL REFERENCES peers(id),
    vote_type TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB,
    UNIQUE(market_data_id, peer_id)
);

CREATE INDEX IF NOT EXISTS idx_votes_market_data ON votes(market_data_id);
CREATE INDEX IF NOT EXISTS idx_votes_peer ON votes(peer_id);
CREATE INDEX IF NOT EXISTS idx_votes_timestamp ON votes(timestamp);