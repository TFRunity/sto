package vehicle

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/vehicle/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	errPlateTaken      = errors.New("plate number already in use")
	errVehicleNotFound = errors.New("vehicle not found")
)

type repository struct {
	queries *db.Queries
}

func (r repository) create(ctx context.Context, ownerID uuid.UUID, details vehicleDetails) (uuid.UUID, error) {
	id, err := r.queries.CreateVehicle(ctx, db.CreateVehicleParams{
		OwnerID: ownerID,
		Number:  details.number,
		Brand:   details.brand,
		Vin:     details.vin,
	})
	if postgres.IsConstraintViolation(err, "23505", "uq_vehicles_number") {
		return uuid.Nil, errPlateTaken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("create vehicle: %w", err)
	}
	return id, nil
}

func (r repository) get(ctx context.Context, id uuid.UUID) (db.GetVehicleRow, error) {
	row, err := r.queries.GetVehicle(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetVehicleRow{}, errVehicleNotFound
	}
	if err != nil {
		return db.GetVehicleRow{}, fmt.Errorf("get vehicle: %w", err)
	}
	return row, nil
}
