package vehicle

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexeBabanov/sto/vehicle/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r repository) replaceCoverPhoto(ctx context.Context, vehicleID, ownerID uuid.UUID, key string) error {
	rows, err := r.queries.ReplaceVehicleCoverPhoto(ctx, db.ReplaceVehicleCoverPhotoParams{
		FileKey:   pgtype.Text{String: key, Valid: true},
		VehicleID: vehicleID,
		OwnerID:   ownerID,
	})
	if err != nil {
		return fmt.Errorf("replace vehicle cover photo: %w", err)
	}
	if rows == 0 {
		return errVehicleNotFound
	}
	return nil
}

func (r repository) addPhoto(ctx context.Context, vehicleID uuid.UUID, orderID *uuid.UUID, key string) (uuid.UUID, error) {
	params := db.AddVehiclePhotoParams{VehicleID: vehicleID, FileKey: key}
	if orderID != nil {
		params.OrderID = pgtype.UUID{Bytes: *orderID, Valid: true}
	}
	id, err := r.queries.AddVehiclePhoto(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errVehicleNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("add vehicle photo: %w", err)
	}
	return id, nil
}

func (r repository) listPhotoIDs(ctx context.Context, vehicleID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.queries.ListVehiclePhotoIDs(ctx, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("list vehicle photos: %w", err)
	}
	return ids, nil
}

func (r repository) getPhotoKey(ctx context.Context, vehicleID, photoID uuid.UUID) (string, error) {
	key, err := r.queries.GetVehiclePhotoKey(ctx, db.GetVehiclePhotoKeyParams{VehicleID: vehicleID, PhotoID: photoID})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errPhotoNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get vehicle photo key: %w", err)
	}
	return key, nil
}
