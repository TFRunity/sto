-- +goose Up
ALTER TABLE orders
    ADD COLUMN intake_engineer_id uuid,
    ADD COLUMN intake_request_id uuid,
    ADD CONSTRAINT ck_orders_intake_request_pair
        CHECK ((intake_engineer_id IS NULL) = (intake_request_id IS NULL)),
    ADD CONSTRAINT uq_orders_intake_request UNIQUE (intake_engineer_id, intake_request_id);
