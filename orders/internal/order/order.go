// Package order implements order creation, item addition and reading.
package order

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type orderItem struct {
	id       uuid.UUID
	itemID   uuid.UUID
	quantity int32
	price    decimal.Decimal
}

func total(items []orderItem) decimal.Decimal {
	sum := decimal.Zero
	for _, item := range items {
		sum = sum.Add(item.price.Mul(decimal.NewFromInt(int64(item.quantity))))
	}
	return sum
}
