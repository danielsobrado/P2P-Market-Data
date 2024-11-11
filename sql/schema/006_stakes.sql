CREATE TABLE IF NOT EXISTS stakes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    peer_id TEXT NOT NULL,
    amount DECIMAL(18,8) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    FOREIGN KEY (peer_id) REFERENCES peers(id)
);

CREATE INDEX IF NOT EXISTS idx_stakes_peer ON stakes(peer_id);
CREATE INDEX IF NOT EXISTS idx_stakes_status ON stakes(status);
CREATE INDEX IF NOT EXISTS idx_stakes_expires_at ON stakes(expires_at);
