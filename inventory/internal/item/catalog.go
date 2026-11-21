package item

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/inventory/internal/api"
	"github.com/AlexeBabanov/sto/inventory/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// ListItems показывает живые товары и их текущие остатки.
func (h *Handler) ListItems(ctx context.Context, _ api.ListItemsRequestObject) (api.ListItemsResponseObject, error) {
	rows, err := h.repo.queries.ListItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("list inventory: %w", err)
	}
	result := make(api.ListItems200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, inventoryEntry(row.ID, row.Name, row.Price, row.Quantity))
	}
	return result, nil
}

// CreateItem создаёт товар с начальным остатком. Ограничения цены и количества проверяются по OpenAPI.
func (h *Handler) CreateItem(ctx context.Context, req api.CreateItemRequestObject) (api.CreateItemResponseObject, error) {
	name, price, err := parseInventoryInput(*req.Body)
	if err != nil {
		return nil, err
	}
	row, err := h.repo.queries.CreateItem(ctx, db.CreateItemParams{Name: name, Price: price, Quantity: int64(req.Body.Quantity)})
	if err != nil {
		return nil, fmt.Errorf("create inventory item: %w", err)
	}
	return api.CreateItem201JSONResponse(inventoryEntry(row.ID, row.Name, row.Price, row.Quantity)), nil
}

// UpdateItem заменяет название, цену и остаток живого товара.
func (h *Handler) UpdateItem(ctx context.Context, req api.UpdateItemRequestObject) (api.UpdateItemResponseObject, error) {
	name, price, err := parseInventoryInput(*req.Body)
	if err != nil {
		return nil, err
	}
	row, err := h.repo.queries.UpdateItem(ctx, db.UpdateItemParams{
		ID: req.Id, Name: name, Price: price, Quantity: int64(req.Body.Quantity),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, itemNotFound()
	}
	if err != nil {
		return nil, fmt.Errorf("update inventory item: %w", err)
	}
	return api.UpdateItem200JSONResponse(inventoryEntry(row.ID, row.Name, row.Price, row.Quantity)), nil
}

// DeleteItem скрывает товар из каталога, сохраняя запись для компенсации списания.
func (h *Handler) DeleteItem(ctx context.Context, req api.DeleteItemRequestObject) (api.DeleteItemResponseObject, error) {
	changed, err := h.repo.queries.DeleteItem(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("delete inventory item: %w", err)
	}
	if changed == 0 {
		return nil, itemNotFound()
	}
	return api.DeleteItem204Response{}, nil
}

func inventoryEntry(id uuid.UUID, name string, price decimal.Decimal, quantity int64) api.InventoryEntry {
	return api.InventoryEntry{Id: id, Name: name, Price: price.StringFixed(2), Quantity: quantity}
}

func parseInventoryInput(input api.InventoryInput) (string, decimal.Decimal, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return "", decimal.Zero, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed", Message: "item name must not be blank",
			Fields: map[string]string{"name": "required"},
		}
	}
	// The OpenAPI middleware has already validated the price format and quantity.
	price, err := decimal.NewFromString(input.Price)
	return name, price, err
}

func itemNotFound() *httpx.Error {
	return &httpx.Error{Status: http.StatusNotFound, Code: "item_not_found", Message: "item not found"}
}
