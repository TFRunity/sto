package httpx

// TODO(dev): фабрика middleware-валидатора запросов по OpenAPI.        → ADR-041
//   - oapi-codegen/nethttp-middleware (kin-openapi); спецификация — из
//     GetSwagger() сгенерированного пакета сервиса (embedded-spec)
//   - обнулить Servers у спецификации: сервис видит путь без /api
//   - no-op аутентификация: JWT уже проверен в Gateway                → ADR-010, ADR-037
//   - ошибки → 400 validation_failed конвертом, fields «поле → причина» → ADR-032
//   - обработчики RequestErrorHandlerFunc / ResponseErrorHandlerFunc для
//     NewStrictHandlerWithOptions — тоже конвертом, тоже отсюда
