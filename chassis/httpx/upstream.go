package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GetJSON декодирует тело ответа 200. При ошибке запроса возвращает
// полученный HTTP-статус или 0, если источник не прислал ответ.
func GetJSON[T any](ctx context.Context, client *http.Client, base url.URL, path string, params url.Values) (T, int, error) {
	return GetJSONWithHeaders[T](ctx, client, base, path, params, nil)
}

// GetJSONWithHeaders читает ответ с дополнительными заголовками, заданными вызывающим сервисом.
func GetJSONWithHeaders[T any](ctx context.Context, client *http.Client, base url.URL, path string, params url.Values, headers http.Header) (T, int, error) {
	var value T
	target := base.JoinPath(path)
	target.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return value, 0, fmt.Errorf("build request for %s: %w", path, err)
	}
	if headers != nil {
		req.Header = headers.Clone()
	}
	resp, err := client.Do(req)
	if err != nil {
		return value, 0, fmt.Errorf("read %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return value, resp.StatusCode, fmt.Errorf("%s returned HTTP %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&value); err != nil {
		return value, resp.StatusCode, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, resp.StatusCode, nil
}

// SearchLimit задаёт число совпадений от одного источника по ADR-039.
const SearchLimit = 200

// SearchTooBroad возвращает общий ответ 422 для поиска по слишком широкому запросу.
func SearchTooBroad() *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: "search_too_broad", Message: "narrow the search query"}
}
