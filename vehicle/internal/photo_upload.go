package vehicle

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/google/uuid"
)

const (
	maxPhotoRequestBytes = 7 * 1024 * 1024 / 2
	maxPhotoPixels       = 16_000_000
	maxPhotoSide         = 8192
)

type photoUpload struct {
	jpeg    []byte
	orderID *uuid.UUID
}

// LimitPhotoUpload bounds the full body before OpenAPI or the strict server reads it.
func LimitPhotoUpload(next http.Handler, onError func(http.ResponseWriter, *http.Request, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isPhotoUpload(r) {
			next.ServeHTTP(w, r)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxPhotoRequestBytes)
		defer func() { _ = r.Body.Close() }()
		body, err := io.ReadAll(r.Body)
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			onError(w, r, &httpx.Error{Status: http.StatusRequestEntityTooLarge, Code: "photo_too_large", Message: "photo request exceeds 3.5 MiB"})
		case err != nil:
			onError(w, r, invalidPhotoField("body", "invalid_multipart"))
		default:
			bounded := r.Clone(r.Context())
			bounded.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, bounded)
		}
	})
}

func isPhotoUpload(r *http.Request) bool {
	path, ok := strings.CutPrefix(r.URL.Path, "/vehicles/")
	if !ok {
		return false
	}
	id, action, ok := strings.Cut(path, "/")
	return ok && id != "" && ((r.Method == http.MethodPut && action == "cover_photo") ||
		(r.Method == http.MethodPost && action == "photos"))
}

func readPhotoUpload(reader *multipart.Reader, allowOrderID bool) (photoUpload, error) {
	var upload photoUpload
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return photoUpload{}, invalidPhotoField("body", "invalid_multipart")
		}
		if err := upload.readPart(part, allowOrderID); err != nil {
			return photoUpload{}, err
		}
	}
	if upload.jpeg == nil {
		return photoUpload{}, invalidPhotoField("photo", "required")
	}
	return upload, nil
}

func (u *photoUpload) readPart(part *multipart.Part, allowOrderID bool) error {
	switch part.FormName() {
	case "photo":
		if u.jpeg != nil {
			return invalidPhotoField("photo", "duplicate")
		}
		data, err := io.ReadAll(part)
		if err != nil {
			return invalidPhotoField("photo", "invalid_multipart")
		}
		u.jpeg, err = reencodePhoto(data)
		return err
	case "order_id":
		if !allowOrderID {
			return invalidPhotoField("order_id", "unexpected")
		}
		if u.orderID != nil {
			return invalidPhotoField("order_id", "duplicate")
		}
		data, err := io.ReadAll(io.LimitReader(part, 37))
		if err != nil || len(data) != 36 {
			return invalidPhotoField("order_id", "invalid_uuid")
		}
		id, err := uuid.Parse(string(data))
		if err != nil || id == uuid.Nil {
			return invalidPhotoField("order_id", "invalid_uuid")
		}
		u.orderID = &id
		return nil
	default:
		return invalidPhotoField("body", "unexpected_part")
	}
}

func reencodePhoto(data []byte) ([]byte, error) {
	switch http.DetectContentType(data) {
	case "image/jpeg", "image/png":
	default:
		return nil, invalidPhotoField("photo", "unsupported_format")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, invalidPhotoField("photo", "invalid_image")
	}
	if config.Width <= 0 || config.Height <= 0 ||
		config.Width > maxPhotoSide || config.Height > maxPhotoSide ||
		config.Width*config.Height > maxPhotoPixels {
		return nil, invalidPhotoField("photo", "dimensions_too_large")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, invalidPhotoField("photo", "invalid_image")
	}
	var result bytes.Buffer
	if err := jpeg.Encode(&result, img, nil); err != nil {
		return nil, fmt.Errorf("encode vehicle photo: %w", err)
	}
	return result.Bytes(), nil
}

func invalidPhotoField(field, reason string) *httpx.Error {
	return &httpx.Error{
		Status: http.StatusBadRequest, Code: "validation_failed",
		Message: "invalid photo upload", Fields: map[string]string{field: reason},
	}
}
