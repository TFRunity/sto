-- Справка по оплаченному заказу (ADR-035).
-- Миграции только forward (ADR-024): Down не пишется.

-- +goose Up
CREATE TABLE certificates (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        uuid        NOT NULL UNIQUE REFERENCES orders (id), -- одна справка на заказ
    status          text        NOT NULL CHECK (status IN ('pending', 'completed', 'failed')),
    attempts        integer     NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 3),
    next_attempt_at timestamptz,
    snapshot        jsonb,                                   -- данные на момент сборки
    pdf             bytea,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- Воркер справок берёт pending / failed с next_attempt_at <= now().
CREATE INDEX idx_certificates_due ON certificates (next_attempt_at)
    WHERE status IN ('pending', 'failed') AND next_attempt_at IS NOT NULL;
