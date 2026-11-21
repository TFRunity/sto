package order

import (
	"context"

	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/jackc/pgx/v5"
)

const directorPageSize = 20

// ListOrdersForDirector применяет фильтры до пагинации. Дополнительная строка определяет has_next.
func (h handler) ListOrdersForDirector(ctx context.Context, req api.ListOrdersForDirectorRequestObject) (api.ListOrdersForDirectorResponseObject, error) {
	page := int32(1)
	if req.Params.Page != nil {
		page = *req.Params.Page
	}
	params := db.ListOrdersForDirectorParams{
		PageLimit: directorPageSize + 1, PageOffset: int64(page-1) * directorPageSize,
	}
	if req.Params.Status != nil {
		status := string(*req.Params.Status)
		params.Status = &status
	}
	if req.Params.ClientIds != nil {
		params.ClientIds = *req.Params.ClientIds
	}
	if req.Params.VehicleIds != nil {
		params.VehicleIds = *req.Params.VehicleIds
	}
	rows, err := h.repo.queries.ListOrdersForDirector(ctx, params)
	if err != nil {
		return nil, err
	}
	hasNext := len(rows) > directorPageSize
	if hasNext {
		rows = rows[:directorPageSize]
	}
	return api.ListOrdersForDirector200JSONResponse{
		Orders: directorRows(rows), Page: page, HasNext: hasNext,
	}, nil
}

// GetOrderStats собирает метрики, последние заказы и загрузку ответственных из одного снимка БД.
func (h handler) GetOrderStats(ctx context.Context, _ api.GetOrderStatsRequestObject) (api.GetOrderStatsResponseObject, error) {
	stats, err := h.repo.stats(ctx)
	return api.GetOrderStats200JSONResponse(stats), err
}

func (r repository) stats(ctx context.Context) (api.OrderStatsSource, error) {
	var stats db.GetOrderStatsRow
	var recent []db.ListOrdersForDirectorRow
	var counts []db.CountActiveOrdersByEngineerRow
	err := postgres.InReadSnapshot(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		var err error
		stats, err = q.GetOrderStats(ctx)
		if err != nil {
			return err
		}
		recent, err = q.ListOrdersForDirector(ctx, db.ListOrdersForDirectorParams{PageLimit: 5})
		if err != nil {
			return err
		}
		counts, err = q.CountActiveOrdersByEngineer(ctx)
		return err
	})
	if err != nil {
		return api.OrderStatsSource{}, err
	}
	var result api.OrderStatsSource
	result.Stats = api.OrderStats{
		TodayOrders: stats.TodayOrders, ActiveOrders: stats.ActiveOrders,
		AwaitingPayment: stats.AwaitingPayment, TodayRevenue: stats.TodayRevenue.StringFixed(2),
		RecentOrders: directorRows(recent),
	}
	result.AssignedCounts = make([]api.EngineerCount, 0, len(counts))
	for _, count := range counts {
		result.AssignedCounts = append(result.AssignedCounts, api.EngineerCount{Id: *count.EngineerID, ActiveOrders: count.ActiveOrders})
	}
	return result, nil
}

func directorRows(rows []db.ListOrdersForDirectorRow) []api.DirectorOrder {
	result := make([]api.DirectorOrder, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.DirectorOrder{
			Id: row.ID, ClientId: row.ClientID, VehicleId: row.VehicleID, EngineerId: row.EngineerID,
			Status: row.Status, Stage: row.Stage, CreatedAt: row.CreatedAt.Time, Total: row.Total.StringFixed(2),
		})
	}
	return result
}
