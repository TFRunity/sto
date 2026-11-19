package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func (r repository) create(ctx context.Context, clientID, vehicleID, engineerID uuid.UUID) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin order creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
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
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit order creation: %w", err)
	}
	return id, nil
}

var errOrderNotFound = errors.New("order not found")

func (r repository) addItem(ctx context.Context, orderID uuid.UUID, item orderItem) (uuid.UUID, error) {
	id, err := r.queries.AddOrderItem(ctx, db.AddOrderItemParams{
		OrderID: orderID, ItemID: item.itemID, Quantity: item.quantity, Price: item.price,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "order_items_order_id_fkey" {
		return uuid.Nil, errOrderNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("add order item: %w", err)
	}
	return id, nil
}

func (r repository) get(ctx context.Context, id uuid.UUID) (db.GetOrderRow, []orderItem, error) {
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
	items := make([]orderItem, 0, len(rows))
	for _, item := range rows {
		items = append(items, orderItem{id: item.ID, itemID: item.ItemID, quantity: item.Quantity, price: item.Price})
	}
	return row, items, nil
}
