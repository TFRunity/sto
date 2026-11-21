-- +goose Up
ALTER TABLE order_items ADD COLUMN name text;
