package internal

// TODO(dev): прокси.
//   - httputil.ReverseProxy с Rewrite                               → ADR-022, ADR-037
//   - исходящий запрос собирается заново: allowlist заголовков
//     (Content-Type, Accept, Accept-Language)                       → ADR-037, D-05
//   - X-User-Id / X-User-Role выставляет Gateway; клиентские — затираются
//   - Authorization вниз не уходит
//   - X-Request-Id генерирует Gateway, от клиента не принимается
//   - ошибки сервисов — как есть; недоступен → 502, таймаут → 504     → ADR-032, D-04
