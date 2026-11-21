// Package internal implements the service catalog.
package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/services/internal/api"
	"github.com/AlexeBabanov/sto/services/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(conn db.DBTX) *Handler {
	return &Handler{queries: db.New(conn)}
}

// ListServices возвращает живые услуги в порядке каталога.
func (h *Handler) ListServices(ctx context.Context, _ api.ListServicesRequestObject) (api.ListServicesResponseObject, error) {
	rows, err := h.queries.ListServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	result := make(api.ListServices200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, serviceEntry(row.ID, row.Name, row.Price))
	}
	return result, nil
}

// GetService отдаёт название и цену живой услуги для добавления позиции в Orders.
func (h *Handler) GetService(ctx context.Context, req api.GetServiceRequestObject) (api.GetServiceResponseObject, error) {
	row, err := h.queries.GetService(ctx, req.Id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, serviceNotFound()
	}
	if err != nil {
		return nil, fmt.Errorf("get service: %w", err)
	}
	return api.GetService200JSONResponse(serviceEntry(row.ID, row.Name, row.Price)), nil
}

// CreateService добавляет услугу с текущей ценой каталога.
func (h *Handler) CreateService(ctx context.Context, req api.CreateServiceRequestObject) (api.CreateServiceResponseObject, error) {
	name, price, err := parseServiceInput(*req.Body)
	if err != nil {
		return nil, err
	}
	row, err := h.queries.CreateService(ctx, db.CreateServiceParams{Name: name, Price: price})
	if err != nil {
		return nil, fmt.Errorf("create service: %w", err)
	}
	return api.CreateService201JSONResponse(serviceEntry(row.ID, row.Name, row.Price)), nil
}

// UpdateService меняет название и цену только у живой услуги.
func (h *Handler) UpdateService(ctx context.Context, req api.UpdateServiceRequestObject) (api.UpdateServiceResponseObject, error) {
	name, price, err := parseServiceInput(*req.Body)
	if err != nil {
		return nil, err
	}
	row, err := h.queries.UpdateService(ctx, db.UpdateServiceParams{ID: req.Id, Name: name, Price: price})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, serviceNotFound()
	}
	if err != nil {
		return nil, fmt.Errorf("update service: %w", err)
	}
	return api.UpdateService200JSONResponse(serviceEntry(row.ID, row.Name, row.Price)), nil
}

// DeleteService скрывает услугу из каталога без физического удаления.
func (h *Handler) DeleteService(ctx context.Context, req api.DeleteServiceRequestObject) (api.DeleteServiceResponseObject, error) {
	changed, err := h.queries.DeleteService(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("delete service: %w", err)
	}
	if changed == 0 {
		return nil, serviceNotFound()
	}
	return api.DeleteService204Response{}, nil
}

func serviceEntry(id uuid.UUID, name string, price decimal.Decimal) api.ServiceEntry {
	return api.ServiceEntry{Id: id, Name: name, Price: price.StringFixed(2)}
}

func serviceNotFound() *httpx.Error {
	return &httpx.Error{Status: http.StatusNotFound, Code: "service_not_found", Message: "service not found"}
}

func parseServiceInput(input api.ServiceInput) (string, decimal.Decimal, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return "", decimal.Zero, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed", Message: "service name must not be blank",
			Fields: map[string]string{"name": "required"},
		}
	}
	price, err := decimal.NewFromString(input.Price)
	return name, price, err
}
