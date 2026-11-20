// Package certificate converts an HTML document into PDF through Gotenberg.
package certificate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	endpoint string
	http     *http.Client
}

func NewClient(baseURL url.URL) *Client {
	return &Client{
		endpoint: baseURL.JoinPath("forms", "chromium", "convert", "html").String(),
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Convert sends index.html and the assets referenced by the page in one request.
func (c *Client) Convert(ctx context.Context, html []byte, assets map[string][]byte) ([]byte, error) {
	request, err := conversionRequest(ctx, c.endpoint, html, assets)
	if err != nil {
		return nil, err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	return readPDF(response)
}

func readPDF(response *http.Response) ([]byte, error) {
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gotenberg returned HTTP %d", response.StatusCode)
	}
	pdf, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return nil, errors.New("gotenberg returned a response without a PDF signature")
	}
	return pdf, nil
}

func conversionRequest(ctx context.Context, endpoint string, html []byte, assets map[string][]byte) (*http.Request, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if err := writeFile(form, "index.html", html); err != nil {
		return nil, err
	}
	for name, data := range assets {
		if err := writeFile(form, name, data); err != nil {
			return nil, err
		}
	}
	if err := form.Close(); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", form.FormDataContentType())
	return request, nil
}

func writeFile(form *multipart.Writer, filename string, data []byte) error {
	part, err := form.CreateFormFile("files", filename)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}
