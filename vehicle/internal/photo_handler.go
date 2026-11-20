package vehicle

import (
	"context"
	"errors"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/vehicle/internal/api"
	"github.com/google/uuid"
)

func (h *Handler) ReplaceVehicleCoverPhoto(ctx context.Context, req api.ReplaceVehicleCoverPhotoRequestObject) (api.ReplaceVehicleCoverPhotoResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	upload, err := readPhotoUpload(req.Body, false)
	if err != nil {
		return nil, err
	}
	err = h.photos.save(upload.jpeg, func(key string) error {
		return h.repo.replaceCoverPhoto(ctx, req.Id, user.ID, key)
	})
	if err != nil {
		return nil, photoWriteError(err)
	}
	return api.ReplaceVehicleCoverPhoto200Response{}, nil
}

func (h *Handler) AddVehiclePhoto(ctx context.Context, req api.AddVehiclePhotoRequestObject) (api.AddVehiclePhotoResponseObject, error) {
	if _, ok := httpx.IdentityFromContext(ctx); !ok {
		return nil, httpx.Unauthorized()
	}
	upload, err := readPhotoUpload(req.Body, true)
	if err != nil {
		return nil, err
	}
	var photoID uuid.UUID
	err = h.photos.save(upload.jpeg, func(key string) error {
		id, err := h.repo.addPhoto(ctx, req.Id, upload.orderID, key)
		photoID = id
		return err
	})
	if err != nil {
		return nil, photoWriteError(err)
	}
	return api.AddVehiclePhoto201JSONResponse{PhotoId: photoID}, nil
}

func photoWriteError(err error) error {
	if errors.Is(err, errVehicleNotFound) && !errors.Is(err, errPhotoCleanup) {
		return vehicleNotFound()
	}
	return err
}

func (h *Handler) GetVehicleCoverPhoto(ctx context.Context, req api.GetVehicleCoverPhotoRequestObject) (api.GetVehicleCoverPhotoResponseObject, error) {
	row, err := h.accessibleVehicle(ctx, req.Id)
	if err != nil {
		return nil, photoReadError(err)
	}
	if !row.CoverPhotoUrl.Valid {
		return nil, photoReadError(errPhotoNotFound)
	}
	jpeg, err := h.openPhoto(row.CoverPhotoUrl.String)
	if err != nil {
		return nil, err
	}
	return api.GetVehicleCoverPhoto200ImagejpegResponse{VehiclePhotoJPEGImagejpegResponse: jpeg}, nil
}

func (h *Handler) GetVehiclePhoto(ctx context.Context, req api.GetVehiclePhotoRequestObject) (api.GetVehiclePhotoResponseObject, error) {
	if _, err := h.accessibleVehicle(ctx, req.Id); err != nil {
		return nil, photoReadError(err)
	}
	key, err := h.repo.getPhotoKey(ctx, req.Id, req.PhotoId)
	if err != nil {
		return nil, photoReadError(err)
	}
	jpeg, err := h.openPhoto(key)
	if err != nil {
		return nil, err
	}
	return api.GetVehiclePhoto200ImagejpegResponse{VehiclePhotoJPEGImagejpegResponse: jpeg}, nil
}

func (h *Handler) openPhoto(key string) (api.VehiclePhotoJPEGImagejpegResponse, error) {
	file, size, err := h.photos.open(key)
	if err != nil {
		return api.VehiclePhotoJPEGImagejpegResponse{}, photoReadError(err)
	}
	// Сгенерированный JPEG-ответ закроет file после отправки.
	return api.VehiclePhotoJPEGImagejpegResponse{Body: file, ContentLength: size}, nil
}

func photoReadError(err error) error {
	if errors.Is(err, errPhotoNotFound) || errors.Is(err, errVehicleNotFound) {
		return &httpx.Error{Status: http.StatusNotFound, Code: "photo_not_found", Message: "photo not found"}
	}
	return err
}
