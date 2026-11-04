-- Seed сотрудников: engineer и director заводятся в БД, не через API (ADR-019).
-- Пароли — только для разработки, хеши bcrypt cost 12 записаны прямо здесь:
--   +70000000001  engineer  пароль engineer-example
--   +70000000002  director  пароль director-example
-- Сменить пароль на уже созданной БД — новой миграцией с UPDATE (ADR-024).
-- Фиксированные id и телефоны — для scripts/e2e.sh.
-- Миграции только forward (ADR-024): Down не пишется.

-- +goose Up
INSERT INTO users (id, name, phone, password_hash, role, work_started_at) VALUES
    ('00000000-0000-4000-8000-000000000001', 'Инженер (seed)',  '+70000000001',
     '$2a$12$qAJXPLpS2.wK90iasDQaf.mlscXSYUk2AknPRo63TeXaUfcKX1Xmm', 'engineer', '2026-01-01'),
    ('00000000-0000-4000-8000-000000000002', 'Директор (seed)', '+70000000002',
     '$2a$12$dP3p4SHnXrkTcUStqAb2P.BwbtIIi/Svkp1tu/oBYDrksti3RH4HS', 'director', '2026-01-01');
