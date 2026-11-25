-- +goose Up
ALTER TABLE users ADD COLUMN notifications_enabled boolean NOT NULL DEFAULT false;
