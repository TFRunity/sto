package order

import (
	"context"

	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	openapiTypes "github.com/oapi-codegen/runtime/types"
)

func (h handler) ListServiceBookings(ctx context.Context, _ api.ListServiceBookingsRequestObject) (api.ListServiceBookingsResponseObject, error) {
	rows, err := h.repo.listServiceBookings(ctx)
	if err != nil {
		return nil, err
	}
	response := make(api.ListServiceBookings200JSONResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, serviceBookingListItem(row))
	}
	return response, nil
}

func serviceBookingListItem(row db.ListServiceBookingsRow) api.ServiceBookingListItem {
	item := api.ServiceBookingListItem{
		Id:        row.ID,
		ClientId:  row.ClientID,
		VehicleId: row.VehicleID,
		Status:    api.ServiceBookingListItemStatusRequested,
		Service:   row.RequestedService,
		Comment:   row.Comment,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.DesiredDate.Valid {
		item.Date = &openapiTypes.Date{Time: row.DesiredDate.Time}
	}
	return item
}
