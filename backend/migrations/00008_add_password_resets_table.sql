-- +goose Up
CREATE TABLE password_resets (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);

CREATE INDEX password_resets_user_id_idx ON password_resets(user_id);
CREATE UNIQUE INDEX password_resets_token_hash_idx ON password_resets(token_hash);

-- +goose Down
DROP TABLE password_resets;