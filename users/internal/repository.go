package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AlexeBabanov/sto/users/internal/db"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: db.New(conn)}
}

func (r *Repository) CreateClient(ctx context.Context, u User) (User, error) {
	row, err := r.q.CreateClient(ctx, db.CreateClientParams{
		ID:           u.ID,
		Name:         u.Name,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
	})

	if isPhoneTaken(err) {
		return User{},
			ErrPhoneTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("create client: %w", err)
	}
	return toUser(row), nil
}

func (r *Repository) GetByPhone(ctx context.Context, phone string) (User, error) {
	row, err := r.q.GetUserByPhone(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user by phone: %w", err)
	}
	return toUser(row), nil
}

func isPhoneTaken(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" && pgErr.ConstraintName == "uq_users_phone"
}

func toUser(row db.User) User {
	return User{
		ID:           row.ID,
		Name:         row.Name,
		Phone:        row.Phone,
		PasswordHash: row.PasswordHash,
		Role:         Role(row.Role),
	}
}
