package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var errItemNotFound = errors.New("inventory item not found")

type inventoryClient struct {
	baseURL url.URL
	http    *http.Client
}

func (c inventoryClient) price(ctx context.Context, itemID uuid.UUID) (decimal.Decimal, error) {
	endpoint := c.baseURL.JoinPath("inventory", itemID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("build inventory request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("read inventory price: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return decimal.Zero, errItemNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("inventory returned HTTP %d", resp.StatusCode)
	}

	var item struct {
		Price string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return decimal.Zero, fmt.Errorf("decode inventory item: %w", err)
	}
	price, err := decimal.NewFromString(item.Price)
	if err != nil {
		return decimal.Zero, fmt.Errorf("decode inventory price: %w", err)
	}
	if price.IsNegative() {
		return decimal.Zero, errors.New("inventory returned a negative price")
	}
	return price, nil
}
