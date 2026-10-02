-- +goose Up
CREATE TYPE stock_movement_type AS ENUM ('in', 'out');

CREATE TABLE stock_movements (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    movement_type stock_movement_type NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX stock_movements_product_id_idx ON stock_movements(product_id);
CREATE INDEX stock_movements_user_id_idx ON stock_movements(user_id);
CREATE INDEX stock_movements_created_at_idx ON stock_movements(created_at);

-- +goose Down
DROP TABLE stock_movements;
DROP TYPE stock_movement_type;
