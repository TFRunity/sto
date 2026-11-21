package vehicle

import (
	"context"
	"errors"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/vehicle/internal/api"
	"github.com/AlexeBabanov/sto/vehicle/internal/db"
	"github.com/google/uuid"
)

type Handler struct {
	repo   repository
	photos photoStorage
}

// NewHandler подключает базу ТС и файловое хранилище фото.
func NewHandler(conn db.DBTX, photoDir string) *Handler {
	return &Handler{
		repo:   repository{queries: db.New(conn)},
		photos: photoStorage{dir: photoDir},
	}
}

// CreateVehicle записывает ТС за текущим владельцем после нормализации номера.
func (h *Handler) CreateVehicle(ctx context.Context, req api.CreateVehicleRequestObject) (api.CreateVehicleResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	details, fields := prepareVehicleDetails(req.Body.Number, req.Body.Brand, req.Body.Vin)
	if len(fields) != 0 {
		return nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed",
			Message: "vehicle details must not be blank", Fields: fields,
		}
	}
	id, err := h.repo.create(ctx, user.ID, details)
	if errors.Is(err, errPlateTaken) {
		return nil, &httpx.Error{Status: http.StatusConflict, Code: "plate_taken", Message: errPlateTaken.Error()}
	}
	if err != nil {
		return nil, err
	}
	return api.CreateVehicle201JSONResponse{VehicleId: id}, nil
}

// GetVehicle собирает карточку из данных ТС и ссылок на доступные фото.
func (h *Handler) GetVehicle(ctx context.Context, req api.GetVehicleRequestObject) (api.GetVehicleResponseObject, error) {
	row, err := h.accessibleVehicle(ctx, req.Id)
	if errors.Is(err, errVehicleNotFound) {
		return nil, vehicleNotFound()
	}
	if err != nil {
		return nil, err
	}
	photoIDs, err := h.repo.listPhotoIDs(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	base := "/api/vehicles/" + row.ID.String()
	photos := make([]api.VehiclePhotoLink, 0, len(photoIDs))
	for _, id := range photoIDs {
		photos = append(photos, api.VehiclePhotoLink{PhotoId: id, Url: base + "/photos/" + id.String()})
	}
	card := api.GetVehicle200JSONResponse{
		VehicleId: row.ID,
		Number:    row.Number,
		Brand:     row.Brand,
		Vin:       row.Vin,
		Photos:    photos,
	}
	if row.CoverPhotoUrl.Valid {
		url := base + "/cover_photo"
		card.CoverPhotoUrl = &url
	}
	return card, nil
}

func (h *Handler) accessibleVehicle(ctx context.Context, id uuid.UUID) (db.GetVehicleRow, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return db.GetVehicleRow{}, httpx.Unauthorized()
	}
	row, err := h.repo.get(ctx, id)
	if err != nil {
		return db.GetVehicleRow{}, err
	}
	if user.Role == httpx.RoleClient && row.OwnerID != user.ID {
		return db.GetVehicleRow{}, errVehicleNotFound
	}
	return row, nil
}

func vehicleNotFound() *httpx.Error {
	return &httpx.Error{Status: http.StatusNotFound, Code: "vehicle_not_found", Message: errVehicleNotFound.Error()}
}
