// Package saga подтверждает оплату и оркестрирует списание товара.
// Новая отметка сохраняется при отказе склада. Успешное списание ставит
// справку в очередь, а старые попытки v1 завершаются по прежнему дедлайну.
package saga

import "github.com/AlexeBabanov/sto/chassis/stockmessage"

const (
	verifyingStock = "verifying_stock"
	completed      = "completed"
	failed         = "failed"
)

// replyStep вызывается из applyReply (consumer.go). Это таблица переходов:
// пока ждём списания, положительный ответ ведёт в completed, отказ — в failed.
// Пустой результат означает, что для этой пары состояния и ответа перехода нет.
// Проверка срока и запись нового шага происходят дальше, в transition.
func replyStep(current, event string) string {
	if current != verifyingStock {
		return ""
	}
	switch event {
	case stockmessage.StockDeducted:
		return completed
	case stockmessage.StockRejected:
		return failed
	default:
		return ""
	}
}
