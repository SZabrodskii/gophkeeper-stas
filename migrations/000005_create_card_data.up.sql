CREATE TABLE card_data (
    entry_id              UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
    encrypted_number      BYTEA NOT NULL,
    encrypted_expiry      BYTEA NOT NULL,
    encrypted_holder_name BYTEA NOT NULL,
    encrypted_cvv         BYTEA NOT NULL
);
