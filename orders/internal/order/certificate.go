package order

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/certificate"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetCertificate возвращает готовый PDF владельцу заказа. Повторная загрузка читает те же байты.
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

// NewCertificateSource loads order lines and the names needed for the certificate.
func NewCertificateSource(pool *pgxpool.Pool, inventoryURL, usersURL url.URL) certificate.LoadSnapshot {
	source := certificateSource{
		repo:      repository{pool: pool, queries: db.New(pool)},
		inventory: newCatalogClient(inventoryURL, "inventory"),
		usersURL:  usersURL, http: &http.Client{Timeout: 3 * time.Second},
	}
	return source.load
}

type certificateSource struct {
	repo      repository
	inventory catalogClient
	usersURL  url.URL
	http      *http.Client
}

func (s certificateSource) load(ctx context.Context, id uuid.UUID) (certificate.Snapshot, error) {
	snapshot, err := s.repo.get(ctx, id)
	if err != nil {
		return certificate.Snapshot{}, err
	}
	positions, err := s.loadPositions(ctx, snapshot.items)
	if err != nil {
		return certificate.Snapshot{}, err
	}
	for _, service := range snapshot.services {
		positions = append(positions, certificate.Item{Name: service.Name, Quantity: service.Quantity, Price: service.Price.StringFixed(2)})
	}
	clientName, engineers, err := s.loadNames(ctx, id, snapshot.header.ClientID)
	if err != nil {
		return certificate.Snapshot{}, err
	}
	return certificate.Snapshot{
		ClientName: clientName, Engineers: engineers,
		OrderID: id, CreatedAt: snapshot.header.CreatedAt.Time.Format("02.01.2006"), Items: positions, Total: total(snapshot.items, snapshot.services).StringFixed(2),
	}, nil
}

func (s certificateSource) loadPositions(ctx context.Context, items []orderItem) ([]certificate.Item, error) {
	positions := make([]certificate.Item, 0, len(items))
	names := make(map[uuid.UUID]string)
	for _, item := range items {
		name, known := names[item.itemID]
		if item.name != nil {
			name = *item.name
		} else if !known {
			product, err := s.inventory.lookup(ctx, item.itemID)
			if err != nil && !errors.Is(err, errCatalogEntryNotFound) {
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

// loadNames читает список исполнителей и отправляет один запрос Users вне транзакции.
func (s certificateSource) loadNames(ctx context.Context, orderID, clientID uuid.UUID) (string, []string, error) {
	engineers, err := s.repo.queries.ListCertificateEngineerIDs(ctx, orderID)
	if err != nil {
		return "", nil, err
	}
	params := url.Values{"ids": {clientID.String()}}
	for _, id := range engineers {
		params.Add("ids", id.String())
	}
	type person struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}
	people, _, err := httpx.GetJSON[[]person](ctx, s.http, s.usersURL, "/users", params)
	if err != nil {
		return "", nil, err
	}
	names := make(map[uuid.UUID]string, len(people))
	for _, person := range people {
		names[person.ID] = person.Name
	}
	engineerNames := make([]string, 0, len(engineers))
	for _, engineer := range engineers {
		engineerNames = append(engineerNames, displayName(names, engineer))
	}
	return displayName(names, clientID), engineerNames, nil
}

func displayName(names map[uuid.UUID]string, id uuid.UUID) string {
	if name := names[id]; name != "" {
		return name
	}
	// Engineer-created orders can refer to a client ID that Users has never seen.
	return id.String()
}
