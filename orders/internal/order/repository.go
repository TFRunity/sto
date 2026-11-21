package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errOrderConflict     = errors.New("order state does not allow the operation")
	errOrderNotFound     = errors.New("order not found")
	errOrderNotClaimable = errors.New("order cannot be claimed")
)

type repository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func (r repository) create(ctx context.Context, clientID, vehicleID, engineerID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := postgres.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		inserted, err := insertOrderWithEngineer(ctx, db.New(tx), clientID, vehicleID, engineerID)
		id = inserted
		return err
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create order: %w", err)
	}
	return id, nil
}

// Для назначения ответственного обе вставки должны пройти в одной транзакции.
func insertOrderWithEngineer(ctx context.Context, queries *db.Queries, clientID, vehicleID, engineerID uuid.UUID) (uuid.UUID, error) {
	id, err := queries.CreateOrder(ctx, db.CreateOrderParams{
		ClientID: clientID, VehicleID: vehicleID, EngineerID: engineerID,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert order: %w", err)
	}
	if err := queries.AddOrderEngineer(ctx, db.AddOrderEngineerParams{
		OrderID: id, EngineerID: engineerID,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("assign order engineer: %w", err)
	}
	return id, nil
}

func (r repository) addItem(ctx context.Context, orderID uuid.UUID, item orderItem) (uuid.UUID, error) {
	id, err := r.queries.AddOrderItem(ctx, db.AddOrderItemParams{
		OrderID: orderID, ItemID: item.itemID, Name: item.name, Quantity: item.quantity, Price: item.price,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, r.orderConflict(ctx, orderID)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("add order item: %w", err)
	}
	return id, nil
}

type orderSnapshot struct {
	header   db.GetOrderRow
	items    []orderItem
	services []db.ListOrderServicesRow
}

// get reads the order state and both kinds of positions from one database snapshot.
func (r repository) get(ctx context.Context, id uuid.UUID) (orderSnapshot, error) {
	var snapshot orderSnapshot
	var items []db.ListOrderItemsRow
	err := postgres.InReadSnapshot(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)
		var err error
		snapshot.header, err = queries.GetOrder(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return errOrderNotFound
		}
		if err != nil {
			return fmt.Errorf("read order: %w", err)
		}
		items, err = queries.ListOrderItems(ctx, id)
		if err != nil {
			return fmt.Errorf("list order items: %w", err)
		}
		snapshot.services, err = queries.ListOrderServices(ctx, id)
		if err != nil {
			return fmt.Errorf("list order services: %w", err)
		}
		return nil
	})
	if err != nil {
		return orderSnapshot{}, err
	}
	snapshot.items = mapOrderItems(items)
	return snapshot, nil
}

func mapOrderItems(rows []db.ListOrderItemsRow) []orderItem {
	items := make([]orderItem, 0, len(rows))
	for _, item := range rows {
		items = append(items, orderItem{
			id:       item.ID,
			itemID:   item.ItemID,
			name:     item.Name,
			quantity: item.Quantity,
			price:    item.Price,
		})
	}
	return items
}

var errCertificateNotFound = errors.New("certificate not found")

func (r repository) getCertificate(ctx context.Context, orderID, clientID uuid.UUID) ([]byte, error) {
	pdf, err := r.queries.GetCompletedCertificate(ctx, db.GetCompletedCertificateParams{OrderID: orderID, ClientID: clientID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errCertificateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get completed certificate: %w", err)
	}
	return pdf, nil
}

func (r repository) createBooking(ctx context.Context, clientID, vehicleID uuid.UUID, service string, date time.Time, comment *string) (uuid.UUID, error) {
	id, err := r.queries.CreateServiceBooking(ctx, db.CreateServiceBookingParams{
		ClientID: clientID, VehicleID: vehicleID, RequestedService: &service,
		DesiredDate: pgtype.Date{Time: date, Valid: true}, Comment: comment,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create service booking: %w", err)
	}
	return id, nil
}

func (r repository) listServiceBookings(ctx context.Context) ([]db.ListServiceBookingsRow, error) {
	rows, err := r.queries.ListServiceBookings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list service bookings: %w", err)
	}
	return rows, nil
}

func (r repository) claim(ctx context.Context, orderID, engineerID uuid.UUID) error {
	err := postgres.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		changed, err := q.ClaimOrder(ctx, db.ClaimOrderParams{
			OrderID: orderID, EngineerID: engineerID,
		})
		if err != nil {
			return err
		}
		if changed != 0 {
			return q.AddOrderEngineer(ctx, db.AddOrderEngineerParams{
				OrderID: orderID, EngineerID: engineerID,
			})
		}
		_, err = q.GetOrder(ctx, orderID)
		if errors.Is(err, pgx.ErrNoRows) {
			return errOrderNotFound
		}
		if err != nil {
			return err
		}
		return errOrderNotClaimable
	})
	if err != nil {
		return fmt.Errorf("claim order: %w", err)
	}
	return nil
}

func (r repository) listClientOrderHistory(ctx context.Context, clientID uuid.UUID, vehicleID *uuid.UUID) ([]db.ListClientOrderHistoryRow, error) {
	rows, err := r.queries.ListClientOrderHistory(ctx, db.ListClientOrderHistoryParams{
		ClientID:  clientID,
		VehicleID: vehicleID,
	})
	if err != nil {
		return nil, fmt.Errorf("list client order history: %w", err)
	}
	return rows, nil
}

func (r repository) changeStage(ctx context.Context, id uuid.UUID, stage string) error {
	changed, err := r.queries.ChangeOrderStage(ctx, db.ChangeOrderStageParams{ID: id, Stage: stage})
	if err != nil {
		return fmt.Errorf("change order stage: %w", err)
	}
	if changed == 0 {
		return r.orderConflict(ctx, id)
	}
	return nil
}

func (r repository) cancel(ctx context.Context, id uuid.UUID) error {
	changed, err := r.queries.CancelOrder(ctx, id)
	if err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}
	if changed == 0 {
		return r.orderConflict(ctx, id)
	}
	return nil
}

func (r repository) addService(ctx context.Context, orderID, serviceID uuid.UUID, quantity int32, service catalogEntry) (uuid.UUID, error) {
	id, err := r.queries.AddOrderService(ctx, db.AddOrderServiceParams{
		OrderID: orderID, ServiceID: serviceID, Name: service.name, Price: service.price, Quantity: quantity,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, r.orderConflict(ctx, orderID)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("add order service: %w", err)
	}
	return id, nil
}

func (r repository) orderConflict(ctx context.Context, id uuid.UUID) error {
	_, err := r.queries.GetOrder(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return errOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("read order after rejected update: %w", err)
	}
	return errOrderConflict
}
