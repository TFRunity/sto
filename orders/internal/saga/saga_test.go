package saga

import (
	"testing"

	"github.com/AlexeBabanov/sto/chassis/stockmessage"
)

func TestReplyStep(t *testing.T) {
	tests := []struct {
		name    string
		current string
		event   string
		want    string
	}{
		{name: "deducted", current: verifyingStock, event: stockmessage.StockDeducted, want: completed},
		{name: "rejected", current: verifyingStock, event: stockmessage.StockRejected, want: failed},
		{name: "unrelated event", current: verifyingStock, event: stockmessage.CancelStock},
		{name: "duplicate success", current: completed, event: stockmessage.StockDeducted},
		{name: "rejection after success", current: completed, event: stockmessage.StockRejected},
		{name: "late success", current: failed, event: stockmessage.StockDeducted},
		{name: "duplicate rejection", current: failed, event: stockmessage.StockRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replyStep(tt.current, tt.event); got != tt.want {
				t.Errorf("replyStep(%q, %q) = %q, want %q", tt.current, tt.event, got, tt.want)
			}
		})
	}
}
