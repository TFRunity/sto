package order

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/certificate"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (h handler) GetCertificate(ctx context.Context, req api.GetCertificateRequestObject) (api.GetCertificateResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	pdf, err := h.repo.getCertificate(ctx, req.Id, user.ID)
	if errors.Is(err, errCertificateNotFound) {
		return nil, &httpx.Error{Status: http.StatusNotFound, Code: "certificate_not_found", Message: "certificate not found"}
	}
	if err != nil {
		return nil, err
	}
	return api.GetCertificate200ApplicationpdfResponse{Body: bytes.NewReader(pdf), ContentLength: int64(len(pdf))}, nil
}

// NewCertificateSource reuses the order snapshot, stored prices, and Inventory client.
func NewCertificateSource(pool *pgxpool.Pool, inventoryURL url.URL) certificate.LoadSnapshot {
	source := certificateSource{repo: repository{pool: pool, queries: db.New(pool)}, inventory: newInventoryClient(inventoryURL)}
	return source.load
}

type certificateSource struct {
	repo      repository
	inventory inventoryClient
}

func (s certificateSource) load(ctx context.Context, id uuid.UUID) (certificate.Snapshot, error) {
	row, items, err := s.repo.get(ctx, id)
	if err != nil {
		return certificate.Snapshot{}, err
	}
	positions, err := s.loadPositions(ctx, items)
	if err != nil {
		return certificate.Snapshot{}, err
	}
	return certificate.Snapshot{
		OrderID: id, CreatedAt: row.CreatedAt.Time.Format("02.01.2006"), Items: positions, Total: total(items).StringFixed(2),
	}, nil
}

func (s certificateSource) loadPositions(ctx context.Context, items []orderItem) ([]certificate.Item, error) {
	positions := make([]certificate.Item, 0, len(items))
	names := make(map[uuid.UUID]string)
	for _, item := range items {
		name, known := names[item.itemID]
		if !known {
			product, err := s.inventory.lookup(ctx, item.itemID)
			if err != nil && !errors.Is(err, errItemNotFound) {
				return nil, err
			}
			name = product.name
			if name == "" {
				// Inventory may have removed the product since the purchase.
				name = item.itemID.String()
			}
			names[item.itemID] = name
		}
		positions = append(positions, certificate.Item{Name: name, Quantity: item.quantity, Price: item.price.StringFixed(2)})
	}
	return positions, nil
}
