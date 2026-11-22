package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/AlexeBabanov/sto/chassis/config"
)

func TestSecretRedaction(t *testing.T) {
	const raw = "example-private-value"
	secret := config.Secret(raw)
	encoded, err := json.Marshal(secret)
	if err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	slog.New(slog.NewJSONHandler(&log, nil)).Info("test", "secret", secret)

	outputs := map[string]string{
		"fmt":  fmt.Sprintf("%s %v %#v", secret, secret, secret),
		"JSON": string(encoded),
		"slog": log.String(),
	}
	for name, output := range outputs {
		if strings.Contains(output, raw) {
			t.Errorf("%s exposed the secret", name)
		}
	}
	if secret.Reveal() != raw {
		t.Fatal("Reveal did not return the original value")
	}
}
