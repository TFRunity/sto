package order

import (
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
)

// По контракту деньги передаются строкой с двумя знаками после точки.
type itemJSON struct {
	ID       uuid.UUID `json:"id"`
	ItemID   uuid.UUID `json:"item_id"`
	Name     *string   `json:"name,omitempty"`
	Quantity int32     `json:"quantity"`
	Price    string    `json:"price"`
}

func itemResponse(item orderItem) itemJSON {
	return itemJSON{
		ID:       item.id,
		ItemID:   item.itemID,
		Name:     item.name,
		Quantity: item.quantity,
		Price:    item.price.StringFixed(2),
	}
}

func (item itemJSON) VisitAddOrderItemResponse(w http.ResponseWriter) error {
	httpx.WriteJSON(w, http.StatusCreated, item)
	return nil
}

type orderJSON struct {
	ID               uuid.UUID          `json:"id"`
	ClientID         uuid.UUID          `json:"client_id"`
	VehicleID        uuid.UUID          `json:"vehicle_id"`
	EngineerID       *uuid.UUID         `json:"engineer_id,omitempty"`
	Status           api.OrderStatus    `json:"status"`
	Stage            api.OrderStage     `json:"stage"`
	PaymentConfirmed bool               `json:"payment_confirmed"`
	Items            []itemJSON         `json:"items"`
	Services         []api.OrderService `json:"services"`
	Total            string             `json:"total"`
}

func orderResponse(row db.GetOrderRow, items []orderItem, services []db.ListOrderServicesRow) orderJSON {
	positions := make([]itemJSON, 0, len(items))
	for _, item := range items {
		positions = append(positions, itemResponse(item))
	}
	servicePositions := make([]api.OrderService, 0, len(services))
	for _, service := range services {
		servicePositions = append(servicePositions, api.OrderService{
			Id: service.ID, ServiceId: service.ServiceID, Name: service.Name,
			Quantity: service.Quantity, Price: service.Price.StringFixed(2),
		})
	}
	return orderJSON{
		ID:               row.ID,
		ClientID:         row.ClientID,
		VehicleID:        row.VehicleID,
		EngineerID:       row.EngineerID,
		Status:           api.OrderStatus(row.Status),
		Stage:            api.OrderStage(row.Stage),
		PaymentConfirmed: row.PaymentConfirmed,
		Items:            positions,
		Services:         servicePositions,
		Total:            total(items, services).StringFixed(2),
	}
}

func (order orderJSON) VisitGetOrderResponse(w http.ResponseWriter) error {
	httpx.WriteJSON(w, http.StatusOK, order)
	return nil
}
