package order

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

var errVehicleNotFound = errors.New("vehicle not found")

type vehicleClient struct {
	baseURL url.URL
	http    *http.Client
}

func newVehicleClient(baseURL url.URL) vehicleClient {
	return vehicleClient{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

func (c vehicleClient) checkOwnership(ctx context.Context, vehicleID, ownerID uuid.UUID) error {
	// ownerID взят из доверенной идентичности Gateway и передаётся как параметр внутреннего API.
	endpoint := c.baseURL.JoinPath("internal", "vehicles", vehicleID.String(), "ownership")
	query := endpoint.Query()
	query.Set("owner_id", ownerID.String())
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("build vehicle ownership request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("check vehicle ownership: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return errVehicleNotFound
	default:
		return fmt.Errorf("vehicle ownership check returned HTTP %d", resp.StatusCode)
	}
}
