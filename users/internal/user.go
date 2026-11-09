// Package internal — домен пользователей (один домен — плоская структура, ADR-004).
package internal

import "github.com/google/uuid"

type Role string

const (
	RoleClient   Role = "client"
	RoleEngineer Role = "engineer"
	RoleDirector Role = "director"
)

type User struct {
	ID           uuid.UUID
	Name         string
	Phone        string
	PasswordHash string
	Role         Role
}
