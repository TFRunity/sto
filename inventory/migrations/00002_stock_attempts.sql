-- Попытки списания по саге оплаты — идемпотентность Inventory (ADR-030, D-02).
-- Миграции только forward (ADR-024): Down не пишется.

-- +goose Up
CREATE TABLE stock_attempts (
    saga_id    uuid        PRIMARY KEY,                       -- ключ идемпотентности
    order_id   uuid        NOT NULL,                          -- для логов; заказ в Orders, без FK
    state      text        NOT NULL CHECK (state IN ('deducted', 'rejected', 'cancelled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Что и сколько списано — источник истины для возврата по CancelStock.
CREATE TABLE stock_attempt_items (
    saga_id  uuid    NOT NULL REFERENCES stock_attempts (saga_id),
    item_id  uuid    NOT NULL REFERENCES inventory (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (saga_id, item_id)
);
