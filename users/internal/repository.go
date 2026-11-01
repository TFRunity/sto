package internal

// TODO(dev): репозиторий пользователей (поверх sqlc).
//   - создание пользователя; занятый телефон → доменная ошибка
//     (→ 409 phone_taken)                                            → sequence-register
//   - поиск по телефону для входа                                   → sequence-engineer-login
//   - каждый запрос фильтрует deleted_at IS NULL                     → ADR-033
//   - sqlc-модели не выходят за пределы репозитория                  → ADR-024
