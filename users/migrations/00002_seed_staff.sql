-- Seed сотрудников: engineer и director заводятся в БД, не через API (ADR-019).
-- Хеши bcrypt — из окружения сервиса (.env → docker-compose.yml / make run),
-- подставляются goose через ENVSUB. Миграция применяется один раз: смена хеша в .env
-- на уже созданную БД не влияет (новая миграция или пересоздание тома).
-- Фиксированные id и телефоны — для scripts/e2e.sh.
-- Миграции только forward (ADR-024): Down не пишется.

-- +goose Up
-- +goose ENVSUB ON
INSERT INTO users (id, name, phone, password_hash, role, work_started_at) VALUES
    ('00000000-0000-4000-8000-000000000001', 'Инженер (seed)',  '+70000000001',
     '${SEED_ENGINEER_PASSWORD_HASH}', 'engineer', '2026-01-01'),
    ('00000000-0000-4000-8000-000000000002', 'Директор (seed)', '+70000000002',
     '${SEED_DIRECTOR_PASSWORD_HASH}', 'director', '2026-01-01');
-- +goose ENVSUB OFF
