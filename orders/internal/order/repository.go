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
		OrderID: orderID, ItemID: item.itemID, Quantity: item.quantity, Price: item.price,
	})
	if postgres.IsConstraintViolation(err, "23503", "order_items_order_id_fkey") {
		return uuid.Nil, errOrderNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("add order item: %w", err)
	}
	return id, nil
}

func (r repository) get(ctx context.Context, id uuid.UUID) (db.GetOrderRow, []orderItem, error) {
	// Заказ, подтверждение оплаты и позиции читаем из одного снимка.
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return db.GetOrderRow{}, nil, fmt.Errorf("begin order read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
	row, err := queries.GetOrder(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetOrderRow{}, nil, errOrderNotFound
	}
	if err != nil {
		return db.GetOrderRow{}, nil, fmt.Errorf("read order: %w", err)
	}
	rows, err := queries.ListOrderItems(ctx, id)
	if err != nil {
		return db.GetOrderRow{}, nil, fmt.Errorf("list order items: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return db.GetOrderRow{}, nil, fmt.Errorf("commit order read: %w", err)
	}
	return row, mapOrderItems(rows), nil
}

func mapOrderItems(rows []db.ListOrderItemsRow) []orderItem {
	items := make([]orderItem, 0, len(rows))
	for _, item := range rows {
		items = append(items, orderItem{
			id:       item.ID,
			itemID:   item.ItemID,
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
