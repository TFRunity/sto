// Package stockmessage defines the shared Orders–Inventory message contract.
package stockmessage

import "github.com/google/uuid"

const (
	DeductStock   = "DeductStock"
	CancelStock   = "CancelStock"
	StockDeducted = "StockDeducted"
	StockRejected = "StockRejected"
)

type Attempt struct {
	SagaID  uuid.UUID `json:"saga_id"`
	OrderID uuid.UUID `json:"order_id"`
}

type Deduction struct {
	Attempt
	Items []Item `json:"items"`
}

type Item struct {
	ID       uuid.UUID `json:"item_id"`
	Quantity int64     `json:"qty"`
}
