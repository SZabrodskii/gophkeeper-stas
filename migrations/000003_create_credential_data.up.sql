CREATE TABLE credential_data (
    entry_id           UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
    encrypted_login    BYTEA NOT NULL,
    encrypted_password BYTEA NOT NULL
);