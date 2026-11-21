-- +goose Up
-- A catalogue update can set stock to the int32 limit before a saga returns stock.
ALTER TABLE inventory ALTER COLUMN quantity TYPE bigint;
ALTER TABLE stock_attempt_items ALTER COLUMN quantity TYPE bigint;
