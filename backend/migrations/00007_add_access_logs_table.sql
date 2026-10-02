-- +goose Up
CREATE TYPE login_result AS ENUM ('success', 'failure');

CREATE TABLE access_logs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    result login_result NOT NULL,
    ip_address INET NOT NULL
);

CREATE INDEX access_logs_user_id_idx ON access_logs(user_id);
CREATE INDEX access_logs_attempted_at_idx ON access_logs(attempted_at);

-- +goose Down
DROP TABLE access_logs;
DROP TYPE login_result;