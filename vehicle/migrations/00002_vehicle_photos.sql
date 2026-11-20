-- +goose Up
ALTER TABLE vehicles ADD COLUMN cover_photo_url text;

CREATE TABLE vehicle_photos (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id uuid        NOT NULL REFERENCES vehicles (id),
    order_id   uuid,
    file_key   text        NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_vehicle_photos_vehicle_id ON vehicle_photos (vehicle_id);
