-- +goose Up
CREATE TABLE vehicles (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   uuid        NOT NULL,
    number     text        NOT NULL,
    brand      text        NOT NULL,
    vin        text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_vehicles_number ON vehicles (number) WHERE deleted_at IS NULL;
