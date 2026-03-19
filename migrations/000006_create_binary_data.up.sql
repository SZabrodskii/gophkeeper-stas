CREATE TABLE binary_data (
    entry_id          UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
    encrypted_data    BYTEA NOT NULL,
    original_filename TEXT NOT NULL DEFAULT ''
);
