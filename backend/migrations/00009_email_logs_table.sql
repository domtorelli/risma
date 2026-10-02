-- +goose Up
CREATE TYPE email_type AS ENUM ('new_user', 'low_stock', 'password_reset');

CREATE TYPE email_status AS ENUM ('sent', 'failed');

CREATE TABLE email_logs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    recipient_email VARCHAR(254) NOT NULL,
    email_type email_type NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status email_status NOT NULL,
);


-- +goose Down
DROP TABLE email_logs;
DROP TYPE email_type;
DROP TYPE email_status;
