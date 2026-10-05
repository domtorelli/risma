-- +goose Up
INSERT INTO roles (name) VALUES ('admin'),('operatore');

-- +goose Down
DELETE FROM roles WHERE name IN ('admin', 'operatore');
