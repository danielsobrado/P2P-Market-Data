-- sql/schema/004_scripts.sql
CREATE TABLE IF NOT EXISTS scripts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    hash TEXT NOT NULL,
    data_type TEXT NOT NULL,
    author TEXT,
    version TEXT,
    dependencies TEXT[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, version)
);

CREATE INDEX IF NOT EXISTS idx_scripts_name_version ON scripts(name, version);
CREATE INDEX IF NOT EXISTS idx_scripts_data_type ON scripts(data_type);