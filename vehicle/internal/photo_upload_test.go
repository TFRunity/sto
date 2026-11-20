package vehicle

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlexeBabanov/sto/chassis/httpx"
)

func TestReencodePhoto(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	var pngBody, jpegBody, oversizedPNG bytes.Buffer
	if err := png.Encode(&pngBody, img); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&jpegBody, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&oversizedPNG, image.NewGray(image.Rect(0, 0, maxPhotoSide+1, 1))); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name   string
		data   []byte
		reason string
	}{
		{"PNG", pngBody.Bytes(), ""},
		{"JPEG", jpegBody.Bytes(), ""},
		{"text", []byte("not an image"), "unsupported_format"},
		{"oversized dimensions", oversizedPNG.Bytes(), "dimensions_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := reencodePhoto(tc.data)
			if tc.reason != "" {
				var apiErr *httpx.Error
				if !errors.As(err, &apiErr) || apiErr.Fields["photo"] != tc.reason {
					t.Fatalf("expected %s, got %v", tc.reason, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			config, err := jpeg.DecodeConfig(bytes.NewReader(output))
			if err != nil || config.Width != 3 || config.Height != 2 {
				t.Fatalf("unexpected JPEG: %v, %v", config, err)
			}
		})
	}
}

func TestLimitPhotoUpload(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/vehicles/example/photos", bytes.NewReader(make([]byte, maxPhotoRequestBytes+1)))
	req.ContentLength = -1
	got := httptest.NewRecorder()
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("oversized request reached handler") })
	onError := func(w http.ResponseWriter, _ *http.Request, err error) {
		var apiErr *httpx.Error
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected API error, got %v", err)
		}
		w.WriteHeader(apiErr.Status)
	}
	LimitPhotoUpload(next, onError).ServeHTTP(got, req)
	if got.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", got.Code)
	}
}
