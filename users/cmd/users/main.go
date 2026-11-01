package main

// TODO(dev): сборка Users.
//   - конфиг (chassis/config): HTTP_ADDR, DATABASE_URL, JWT_SECRET, JWT_TTL
//   - pgxpool; миграции goose при старте (go:embed)                  → ADR-024
//   - репозиторий → сервис → хендлеры; middleware идентичности
//   - HTTP-сервер + graceful shutdown (chassis/httpx)
func main() {
	panic("TODO(dev): users")
}
