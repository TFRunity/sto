-- include-deleted: исторический состав сотрудников включает удалённых после начала отчётного периода.
-- name: ListReportEngineers :many
SELECT id AS engineer_id, role, work_started_at
FROM users
WHERE role = 'engineer'
    AND (work_started_at IS NULL OR work_started_at < sqlc.arg(period_end_date)::date)
    AND (deleted_at IS NULL OR deleted_at > sqlc.arg(period_from)::timestamptz)
ORDER BY id;
