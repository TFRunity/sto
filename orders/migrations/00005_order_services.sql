-- +goose Up
CREATE TABLE order_services (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id uuid NOT NULL REFERENCES orders (id),
    service_id uuid NOT NULL,
    name text NOT NULL,
    quantity integer NOT NULL CHECK (quantity > 0),
    price numeric(12,2) NOT NULL CHECK (price >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_order_services_order_id ON order_services (order_id);
