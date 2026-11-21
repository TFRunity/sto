package postgres_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsConstraintViolation(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"wrapped match", fmt.Errorf("insert: %w", &pgconn.PgError{Code: "23505", ConstraintName: "uq_example"}), true},
		{"other constraint", &pgconn.PgError{Code: "23505", ConstraintName: "uq_other"}, false},
		{"other SQLSTATE", &pgconn.PgError{Code: "23503", ConstraintName: "uq_example"}, false},
		{"unrelated error", errors.New("insert failed"), false},
		{"no error", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := postgres.IsConstraintViolation(tc.err, "23505", "uq_example"); got != tc.want {
				t.Fatalf("got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsPattern(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"А123", "А123"},
		{"a!_%", "a!!!_!%"},
	} {
		if got := postgres.ContainsPattern(tc.input); got != tc.want {
			t.Errorf("ContainsPattern(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
