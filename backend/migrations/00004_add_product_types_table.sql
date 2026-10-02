-- +goose Up
CREATE TABLE product_types (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX product_types_name_lower_idx ON product_types (LOWER(name));

CREATE TRIGGER product_types_set_updated_at
BEFORE UPDATE ON product_types
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

INSERT INTO product_types (name) VALUES ('Buste'), ('Carta'), ('Toner');

-- +goose Down
DROP TABLE product_types;