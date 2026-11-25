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

// GetCertificate возвращает готовый PDF владельцу или директору.
func (h handler) GetCertificate(ctx context.Context, req api.GetCertificateRequestObject) (api.GetCertificateResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	if user.Role != httpx.RoleClient && user.Role != directorRole {
		return nil, certificateRoleForbidden()
	}
	pdf, err := h.repo.getCertificate(ctx, req.Id, user.ID, user.Role == directorRole)
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
		people:    peopleDirectory{baseURL: usersURL, http: &http.Client{Timeout: 3 * time.Second}},
	}
	return source.load
}

type certificateSource struct {
	repo      repository
	inventory catalogClient
	people    peopleDirectory
}

func (s certificateSource) load(ctx context.Context, id uuid.UUID) (certificate.Snapshot, error) {
	snapshot, err := s.repo.get(ctx, id, orderRead{includeEngineers: true})
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
	clientName, engineers, err := s.loadNames(ctx, snapshot.header.ClientID, snapshot.engineerIDs)
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

// loadNames разрешает имена клиента и всех исполнителей одним запросом к Users вне транзакции.
func (s certificateSource) loadNames(ctx context.Context, clientID uuid.UUID, engineers []uuid.UUID) (string, []string, error) {
	ids := make([]uuid.UUID, 0, len(engineers)+1)
	ids = append(ids, clientID)
	ids = append(ids, engineers...)
	names, err := s.people.names(ctx, ids)
	if err != nil {
		return "", nil, err
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
