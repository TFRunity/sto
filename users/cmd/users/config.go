package main

import (
	"time"

	"github.com/AlexeBabanov/sto/chassis/config"
)

type Config struct {
	config.Base

	DatabaseURL config.Secret `env:"DATABASE_URL,notEmpty"`
	JWTSecret   config.Secret `env:"JWT_SECRET,notEmpty"`
	JWTTTL      time.Duration `env:"JWT_TTL" envDefault:"168h"`
}
