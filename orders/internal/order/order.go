// Package order implements order creation, item addition and reading.
package order

import (
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type orderItem struct {
	id       uuid.UUID
	itemID   uuid.UUID
	name     *string
	quantity int32
	price    decimal.Decimal
}

func total(items []orderItem, services []db.ListOrderServicesRow) decimal.Decimal {
	sum := decimal.Zero
	for _, item := range items {
		lineTotal := item.price.Mul(decimal.NewFromInt(int64(item.quantity)))
		sum = sum.Add(lineTotal)
	}
	for _, service := range services {
		sum = sum.Add(service.Price.Mul(decimal.NewFromInt(int64(service.Quantity))))
	}
	return sum
}
