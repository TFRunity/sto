package vehicle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/vehicle/internal/api"
	"github.com/AlexeBabanov/sto/vehicle/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(conn db.DBTX) *Handler {
	return &Handler{queries: db.New(conn)}
}

func (h *Handler) CreateVehicle(ctx context.Context, req api.CreateVehicleRequestObject) (api.CreateVehicleResponseObject, error) {
	user, ok := ctx.Value(identityKey{}).(identity)
	if !ok {
		return nil, unauthorized()
	}
	if user.role != "client" {
		return nil, &httpx.Error{Status: http.StatusForbidden, Code: "forbidden", Message: "access denied"}
	}

	number := normalizePlate(req.Body.Number)
	brand := strings.TrimSpace(req.Body.Brand)
	vin := strings.TrimSpace(req.Body.Vin)
	if number == "" || brand == "" || vin == "" {
		return nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed",
			Message: "number, brand and vin must not be blank",
		}
	}
	id, err := h.queries.CreateVehicle(ctx, db.CreateVehicleParams{
		OwnerID: user.id,
		Number:  number,
		Brand:   brand,
		Vin:     vin,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_vehicles_number" {
		return nil, &httpx.Error{Status: http.StatusConflict, Code: "plate_taken", Message: "plate number already in use"}
	}
	if err != nil {
		return nil, fmt.Errorf("create vehicle: %w", err)
	}
	return api.CreateVehicle201JSONResponse{VehicleId: id}, nil
}

func (h *Handler) GetVehicle(ctx context.Context, req api.GetVehicleRequestObject) (api.GetVehicleResponseObject, error) {
	user, ok := ctx.Value(identityKey{}).(identity)
	if !ok {
		return nil, unauthorized()
	}
	if user.role != "client" && user.role != "engineer" {
		return nil, &httpx.Error{Status: http.StatusForbidden, Code: "forbidden", Message: "access denied"}
	}

	row, err := h.queries.GetVehicle(ctx, req.Id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get vehicle: %w", err)
	}
	if errors.Is(err, pgx.ErrNoRows) || (user.role == "client" && row.OwnerID != user.id) {
		return nil, &httpx.Error{Status: http.StatusNotFound, Code: "vehicle_not_found", Message: "vehicle not found"}
	}

	return api.GetVehicle200JSONResponse{
		VehicleId: row.ID,
		Number:    row.Number,
		Brand:     row.Brand,
		Vin:       row.Vin,
	}, nil
}

type identityKey struct{}

type identity struct {
	id   uuid.UUID
	role string
}

func ReadIdentity(next api.StrictHandlerFunc, _ string) api.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		id, err := uuid.Parse(r.Header.Get(httpx.UserIDHeader))
		role := r.Header.Get(httpx.UserRoleHeader)
		if err != nil || id == uuid.Nil || role == "" {
			return nil, unauthorized()
		}
		return next(context.WithValue(ctx, identityKey{}, identity{id: id, role: role}), w, r, request)
	}
}

func unauthorized() *httpx.Error {
	return &httpx.Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "missing request identity"}
}
