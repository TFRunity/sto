package certificate

import (
	"bytes"
	"context"
	_ "embed"
	"html/template"
	"time"

	"github.com/google/uuid"
)

type Snapshot struct {
	OrderID   uuid.UUID `json:"order_id"`
	CreatedAt string    `json:"created_at"`
	Items     []Item    `json:"items"`
	Total     string    `json:"total"`
}

type Item struct {
	Name     string `json:"name"`
	Quantity int32  `json:"quantity"`
	Price    string `json:"price"`
}

type LoadSnapshot func(context.Context, uuid.UUID) (Snapshot, error)

type converter interface {
	Convert(context.Context, []byte, map[string][]byte) ([]byte, error)
}

//go:embed templates/index.html
var templateHTML string

//go:embed templates/DejaVuSans.ttf
var font []byte

var page = template.Must(template.New("certificate").Parse(templateHTML))

func render(snapshot Snapshot) ([]byte, error) {
	var html bytes.Buffer
	err := page.Execute(&html, snapshot)
	return html.Bytes(), err
}

func nextAttempt(attempts int32, success bool, now time.Time) (string, time.Time) {
	if success {
		return "completed", time.Time{}
	}
	switch attempts {
	case 1:
		return "failed", now.Add(10 * time.Second)
	case 2:
		return "failed", now.Add(time.Minute)
	default:
		return "failed", time.Time{}
	}
}
