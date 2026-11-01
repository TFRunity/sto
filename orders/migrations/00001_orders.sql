-- Заказы, позиции-товары, исполнители (класс «Заказ», ADR-020, ADR-021, ADR-034).
-- Ссылки на другие сервисы (client_id, vehicle_id, engineer_id, item_id) — uuid без FK
-- (ADR-001, ADR-007). Заказы не удаляются — терминальный статус cancelled (ADR-033).
-- Позиции-услуги (order_services) — неделя 2, отдельной миграцией.
-- Миграции только forward (ADR-024): Down не пишется.

-- +goose Up
CREATE TABLE orders (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id         uuid        NOT NULL,                  -- без проверки (ADR-015)
    vehicle_id        uuid        NOT NULL,
    engineer_id       uuid,                                  -- ответственный; NULL у заявки
    status            text        NOT NULL
                      CHECK (status IN ('requested', 'active', 'paid', 'cancelled')),
    stage             text        NOT NULL DEFAULT 'принят'
                      CHECK (stage IN ('принят', 'в_работе', 'готов', 'выдан')),
    requested_service text,                                  -- из заявки клиента
    desired_date      date,
    comment           text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_client_id   ON orders (client_id);
CREATE INDEX idx_orders_engineer_id ON orders (engineer_id);

-- Цена фиксируется при добавлении позиции (ADR-034).
CREATE TABLE order_items (
    id         uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   uuid          NOT NULL REFERENCES orders (id),
    item_id    uuid          NOT NULL,                       -- товар в Inventory
    quantity   integer       NOT NULL CHECK (quantity > 0),
    price      numeric(12,2) NOT NULL CHECK (price >= 0),
    created_at timestamptz   NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);

-- Все исполнители, включая ответственного (ADR-020): INSERT … ON CONFLICT DO NOTHING.
CREATE TABLE order_engineers (
    order_id    uuid        NOT NULL REFERENCES orders (id),
    engineer_id uuid        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id, engineer_id)
);
