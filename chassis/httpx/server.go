package httpx

// TODO(dev): запуск HTTP-сервера и graceful shutdown.
//   - stdlib net/http + ServeMux                                     → ADR-022
//   - GET /healthz — только «процесс жив», всегда 200                → ADR-038, I-09
//   - SIGINT/SIGTERM → Shutdown с таймаутом; воркеры и консьюмеры
//     останавливаются через отмену ctx
