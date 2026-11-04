package main

import (
	"fmt"
	"os"

	"github.com/AlexeBabanov/sto/chassis/config"
	"github.com/AlexeBabanov/sto/chassis/logx"
)

// TODO(dev): сборка Users.
//   - pgxpool; миграции goose при старте (go:embed)                  → ADR-024
//   - репозиторий → сервис → хендлеры; middleware идентичности
//   - HTTP-сервер + graceful shutdown (chassis/httpx)
func main() {
	cfg, err := config.Load[Config]()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger := logx.New("users", cfg.LogLevel)
	logger.Info("users started",
		"addr", cfg.HTTPAddr,
		"log_level", cfg.LogLevel,
		"jwt_ttl", cfg.JWTTTL.String(),
	)
}
