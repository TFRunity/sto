package internal

// TODO(dev): HTTP.
//   - реализация api.StrictServerInterface (internal/api/api.gen.go)  → ADR-028
//   - регистрация и вход недели 1
//   - структура запроса — валидатор по спецификации (chassis/httpx);
//     Validate() — только правила вне спецификации (≤ 72 байт пароля) → ADR-041
//   - ошибки — конвертом chassis/httpx                               → ADR-032
