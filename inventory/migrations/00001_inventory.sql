-- Товары с остатком (класс «Товар», ADR-030, ADR-033, ADR-026).
-- Миграции только forward (ADR-024): Down не пишется.

-- +goose Up
CREATE TABLE inventory (
    id         uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text          NOT NULL,
    price      numeric(12,2) NOT NULL CHECK (price >= 0),
    quantity   integer       NOT NULL CHECK (quantity >= 0),   -- списание WHERE quantity >= N (D-09)
    created_at timestamptz   NOT NULL DEFAULT now(),
    updated_at timestamptz   NOT NULL DEFAULT now(),
    deleted_at timestamptz                                     -- soft delete
);
