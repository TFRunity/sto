-- +goose Up
CREATE TABLE push_devices (
    token         text        PRIMARY KEY,
    user_id       uuid        NOT NULL REFERENCES users (id),
    registered_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_push_devices_user_id ON push_devices (user_id);
