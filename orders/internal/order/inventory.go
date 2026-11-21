package order

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var errCatalogEntryNotFound = errors.New("catalog entry not found")

// A position stores the catalog price in a numeric(12,2) column.
var catalogPricePattern = regexp.MustCompile(`^[0-9]{1,10}[.][0-9]{2}$`)

type catalogClient struct {
	baseURL url.URL
	path    string
	http    *http.Client
}

type catalogEntry struct {
	name  string
	price decimal.Decimal
}

func newCatalogClient(url url.URL, path string) catalogClient {
	return catalogClient{baseURL: url, path: path, http: &http.Client{Timeout: 5 * time.Second}}
}

func (c catalogClient) lookup(ctx context.Context, itemID uuid.UUID) (catalogEntry, error) {
	path := c.path + "/" + itemID.String()
	type catalogResponse struct {
		Name  string `json:"name"`
		Price string `json:"price"`
	}
	item, status, err := httpx.GetJSON[catalogResponse](ctx, c.http, c.baseURL, path, c.baseURL.Query())
	if status == http.StatusNotFound {
		return catalogEntry{}, errCatalogEntryNotFound
	}
	if err != nil {
		return catalogEntry{}, err
	}
	if !catalogPricePattern.MatchString(item.Price) {
		return catalogEntry{}, errors.New("catalog price must fit numeric(12,2)")
	}
	price, err := decimal.NewFromString(item.Price)
	return catalogEntry{name: item.Name, price: price}, err
}
