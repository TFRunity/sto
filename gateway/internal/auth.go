// Package internal — периметр: проверка JWT, роли, проксирование (ADR-010).
package internal

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/AlexeBabanov/sto/chassis/httpx"
)

var ErrUnauthorized = &httpx.Error{
	Status: http.StatusUnauthorized,
	Code:   "unauthorized", Message: "unauthorized",
}

type Identity struct {
	UserID string
	Role   string
}

type tokenClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func Authenticate(authHeader string, secret []byte) (Identity, error) {
	rawToken, err := bearerToken(authHeader)
	if err != nil {
		return Identity{}, err
	}
	claims, err := verifiedClaims(rawToken, secret)
	if err != nil {
		return Identity{}, err
	}
	return identityFrom(claims)
}

func bearerToken(authHeader string) (string, error) {
	rawToken, found := strings.CutPrefix(authHeader, "Bearer ")
	if !found || rawToken == "" {
		return "", errors.New("no bearer token")
	}
	return rawToken, nil
}

func verifiedClaims(rawToken string, secret []byte) (tokenClaims, error) {
	keyFunc := func(*jwt.Token) (any, error) { return secret, nil }

	var claims tokenClaims
	_, err := jwt.ParseWithClaims(rawToken, &claims, keyFunc,
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(), // токен без срока — отказ
	)
	if err != nil {
		return tokenClaims{}, fmt.Errorf("verify token: %w", err)
	}

	return claims, nil
}

func identityFrom(claims tokenClaims) (Identity, error) {
	if claims.Subject == "" || claims.Role == "" {
		return Identity{}, errors.New("token without sub or role")
	}
	return Identity{UserID: claims.Subject, Role: claims.Role}, nil
}
