package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var errItemNotFound = errors.New("inventory item not found")

// A position stores the inventory price in a numeric(12,2) column.
var inventoryPricePattern = regexp.MustCompile(`^[0-9]{1,10}[.][0-9]{2}$`)

type inventoryClient struct {
	baseURL url.URL
	http    *http.Client
}

type inventoryItem struct {
	name  string
	price decimal.Decimal
}

func newInventoryClient(url url.URL) inventoryClient {
	return inventoryClient{baseURL: url, http: &http.Client{Timeout: 5 * time.Second}}
}

func (c inventoryClient) lookup(ctx context.Context, itemID uuid.UUID) (inventoryItem, error) {
	endpoint := c.baseURL.JoinPath("inventory", itemID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return inventoryItem{}, fmt.Errorf("build inventory request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return inventoryItem{}, fmt.Errorf("read inventory item: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return inventoryItem{}, errItemNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return inventoryItem{}, fmt.Errorf("inventory returned HTTP %d", resp.StatusCode)
	}
	return decodeInventoryItem(resp)
}

func decodeInventoryItem(resp *http.Response) (inventoryItem, error) {
	var item struct {
		Name  string `json:"name"`
		Price string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return inventoryItem{}, fmt.Errorf("decode inventory item: %w", err)
	}
	if !inventoryPricePattern.MatchString(item.Price) {
		return inventoryItem{}, errors.New("inventory price must fit numeric(12,2)")
	}
	price, err := decimal.NewFromString(item.Price)
	return inventoryItem{name: item.Name, price: price}, err
}
