-- Состояние саги оплаты (ADR-030, заменяет SQL из ADR-006; колонки status нет — А-05).
-- Миграции только forward (ADR-024): Down не пишется.

-- +goose Up
CREATE TABLE saga_state (
    saga_id      uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     uuid        NOT NULL REFERENCES orders (id),
    current_step text        NOT NULL
                 CHECK (current_step IN ('verifying_stock', 'completed', 'failed')),
    deadline     timestamptz NOT NULL,                       -- now() + 10 s при старте
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_saga_state_order_id ON saga_state (order_id);
-- Воркер дедлайнов читает только незавершённые саги.
CREATE INDEX idx_saga_state_deadline ON saga_state (deadline)
    WHERE current_step = 'verifying_stock';
