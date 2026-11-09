package internal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost       = 12
	maxPasswordBytes = 72
)

type UserStore interface {
	CreateClient(ctx context.Context, u User) (User, error)
	GetByPhone(ctx context.Context, phone string) (User, error)
}

type Service struct {
	users     UserStore
	jwtSecret []byte
	jwtTTL    time.Duration
	dummyHash []byte
}

func NewService(users UserStore, jwtSecret []byte, jwtTTL time.Duration) (*Service, error) {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("dummy hash: %w", err)
	}
	return &Service{users: users, jwtSecret: jwtSecret, jwtTTL: jwtTTL, dummyHash: dummyHash}, nil
}

func (s *Service) Register(ctx context.Context, name string, phone string, password string) (uuid.UUID, string, error) {
	if len(password) > maxPasswordBytes {
		return uuid.Nil, "", ErrPasswordTooLong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("hash password: %w", err)
	}
	user, err := s.users.CreateClient(ctx, User{
		ID:           uuid.New(),
		Name:         name,
		Phone:        phone,
		PasswordHash: string(hash),
	})
	if err != nil {
		return uuid.Nil, "", err
	}
	token, err := s.issueToken(user)
	return user.ID, token, err
}

func (s *Service) Login(ctx context.Context, phone string, password string) (string, error) {
	user, err := s.users.GetByPhone(ctx, phone)
	if errors.Is(err, ErrUserNotFound) {
		s.spendSameTime(password)
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if !passwordMatches(user.PasswordHash, password) {
		return "", ErrInvalidCredentials
	}
	return s.issueToken(user)
}

func (s *Service) issueToken(user User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"role": string(user.Role),
		"exp":  time.Now().Add(s.jwtTTL).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return token, nil
}

func (s *Service) spendSameTime(password string) {
	_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
}

func passwordMatches(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
