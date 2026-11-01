// Package internal — периметр: проверка JWT, роли, проксирование (ADR-010).
package internal

// TODO(dev): проверка JWT.
//   - golang-jwt/jwt/v5, HS256, WithValidMethods — только HS256      → ADR-023
//   - exp обязателен; sub = user_id, роль — отдельный claim
//   - нет/битый токен → 401 конвертом chassis                        → ADR-032
