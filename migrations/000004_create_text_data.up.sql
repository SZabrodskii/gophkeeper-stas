CREATE TABLE text_data (
    entry_id          UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
    encrypted_content BYTEA NOT NULL
);
