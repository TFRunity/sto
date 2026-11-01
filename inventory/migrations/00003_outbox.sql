-- Transactional outbox: ответы саги StockDeducted / StockRejected (ADR-005, D-01).
-- Миграции только forward (ADR-024): Down не пишется.

-- +goose Up
CREATE TABLE outbox (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(), -- = AMQP MessageId
    aggregate_id uuid        NOT NULL,                              -- saga_id
    event_type   text        NOT NULL,
    payload      jsonb       NOT NULL,                              -- только id, без ПДн
    created_at   timestamptz NOT NULL DEFAULT now(),
    published    boolean     NOT NULL DEFAULT false,
    published_at timestamptz
);

-- Без него скорость воркера зависит от размера журнала (D-01).
CREATE INDEX idx_outbox_unpublished ON outbox (created_at) WHERE published = false;
