// Package item — чтение товаров каталога.
package item

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/inventory/internal/api"
	"github.com/AlexeBabanov/sto/inventory/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

var errNotFound = errors.New("item not found")

type item struct {
	id    uuid.UUID
	name  string
	price decimal.Decimal
}

type Repository struct {
	queries *db.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{queries: db.New(conn)}
}

func (r *Repository) get(ctx context.Context, id uuid.UUID) (item, error) {
	row, err := r.queries.GetItem(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return item{}, errNotFound
	}
	if err != nil {
		return item{}, fmt.Errorf("get item %s: %w", id, err)
	}

	return item{
		id:    row.ID,
		name:  row.Name,
		price: row.Price,
	}, nil
}

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetItem(ctx context.Context, req api.GetItemRequestObject) (api.GetItemResponseObject, error) {
	product, err := h.repo.get(ctx, req.Id)
	if errors.Is(err, errNotFound) {
		return nil, &httpx.Error{
			Status:  http.StatusNotFound,
			Code:    "item_not_found",
			Message: "item not found",
		}
	}
	if err != nil {
		return nil, err
	}

	return getItemResponse{
		ID:    product.id,
		Name:  product.name,
		Price: product.price.StringFixed(2),
	}, nil
}

type getItemResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Price string    `json:"price"`
}

func (r getItemResponse) VisitGetItemResponse(w http.ResponseWriter) error {
	httpx.WriteJSON(w, http.StatusOK, r)
	return nil
}
