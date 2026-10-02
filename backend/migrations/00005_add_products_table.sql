-- +goose Up
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    product_type_id UUID NOT NULL REFERENCES product_types(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT,
    price_per_unit NUMERIC(10, 2) NOT NULL CHECK (price_per_unit >= 0),
    stock_quantity INT NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    min_stock_threshold INT NOT NULL DEFAULT 0 CHECK (min_stock_threshold >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX products_product_type_id_idx ON products(product_type_id);
CREATE INDEX products_creator_id_idx ON products(creator_id);
CREATE UNIQUE INDEX products_name_lower_idx ON products (LOWER(name)) WHERE deleted_at IS NULL;

CREATE TRIGGER products_set_updated_at
BEFORE UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE products;