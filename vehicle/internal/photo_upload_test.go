package vehicle

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
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
