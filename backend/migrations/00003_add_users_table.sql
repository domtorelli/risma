-- +goose Up
CREATE TYPE user_status AS ENUM ('active', 'blocked');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(254) NOT NULL,
    password_hash TEXT NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    date_of_birth DATE NOT NULL,
    failed_login_attempts SMALLINT NOT NULL DEFAULT 0 CHECK (failed_login_attempts >= 0),
    status user_status NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX users_role_id_idx ON users (role_id);
CREATE UNIQUE INDEX users_username_lower_idx ON users (LOWER(username));
CREATE UNIQUE INDEX users_email_lower_idx ON users (LOWER(email));

CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE users;
DROP TYPE user_status;