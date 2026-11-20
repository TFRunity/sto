package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsConstraintViolation matches both the SQLSTATE and the named constraint.
func IsConstraintViolation(err error, state, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == state && pgErr.ConstraintName == constraint
}
