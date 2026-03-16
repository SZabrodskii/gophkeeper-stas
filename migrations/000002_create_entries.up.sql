CREATE TABLE entries (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_type VARCHAR(20)  NOT NULL CHECK (entry_type IN ('credential', 'text', 'binary', 'card')),
    name       VARCHAR(255) NOT NULL,
    metadata   JSONB,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entries_user_id ON entries(user_id);
CREATE INDEX idx_entries_user_id_updated_at ON entries(user_id, updated_at);