// Package internal — домен пользователей (один домен — плоская структура, ADR-004).
package internal

// TODO(dev): доменные типы и правила.
//   - роль: client | engineer | director                            → ADR-019
//   - регистрация всегда создаёт client; engineer/director — seed
//   - пароль ≤ 72 байт (лимит bcrypt)                                → ADR-023
