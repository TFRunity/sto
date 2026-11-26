-- +goose Up
CREATE TABLE dialogs (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    kind             text        NOT NULL,
    participant_1_id uuid        NOT NULL,
    participant_2_id uuid        NOT NULL,
    order_id         uuid,
    created_at       timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT dialogs_kind_check CHECK (kind IN ('order', 'director_engineer', 'support')),
    CONSTRAINT dialogs_order_check CHECK ((kind = 'order') = (order_id IS NOT NULL)),
    CONSTRAINT dialogs_distinct_participants_check CHECK (participant_1_id <> participant_2_id)
);

-- Ключ сравнивает участников независимо от порядка. NULL для диалогов
-- без заказа участвует в проверке дублей наравне с UUID заказа.
CREATE UNIQUE INDEX uq_dialogs_kind_participants_order ON dialogs (
    kind,
    (LEAST(participant_1_id, participant_2_id)),
    (GREATEST(participant_1_id, participant_2_id)),
    order_id
) NULLS NOT DISTINCT;

CREATE INDEX idx_dialogs_participant_1_id ON dialogs (participant_1_id);
CREATE INDEX idx_dialogs_participant_2_id ON dialogs (participant_2_id);

CREATE TABLE messages (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    dialog_id  uuid        NOT NULL REFERENCES dialogs (id),
    sender_id  uuid        NOT NULL,
    body       text        NOT NULL,
    is_read    boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_dialog_history ON messages (dialog_id, created_at, id);
