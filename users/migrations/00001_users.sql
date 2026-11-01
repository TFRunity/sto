-- Пользователи (класс «Пользователь», ADR-019, ADR-023, ADR-033).
-- Миграции только forward (ADR-024): Down не пишется.

-- +goose Up
CREATE TABLE users (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(), -- генерируется в Go (ADR-026)
    name            text        NOT NULL,
    phone           text        NOT NULL,
    password_hash   text        NOT NULL,                              -- bcrypt, cost 12
    role            text        NOT NULL CHECK (role IN ('client', 'engineer', 'director')),
    work_started_at date,                                              -- только engineer / director
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz                                        -- soft delete
);

-- Телефон уникален только среди живых (ADR-033, А-03 → 409 phone_taken).
CREATE UNIQUE INDEX uq_users_phone ON users (phone) WHERE deleted_at IS NULL;
