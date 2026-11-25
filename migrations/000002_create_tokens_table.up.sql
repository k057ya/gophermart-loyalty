CREATE TABLE tokens (
  user_id BIGINT NOT NULL REFERENCES users (id),
  token_hash BYTEA PRIMARY KEY CHECK (octet_length(token_hash) = 32),
  valid_until TIMESTAMPTZ NOT NULL
);
