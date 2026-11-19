// Package saga ведёт попытку оплаты от запроса клиента до подтверждения
// списания или возврата заказа в active.
//
// Пример — клиент оплачивает заказ с одним масляным фильтром.
//
//  1. Клиент вызывает POST /orders/{id}/pay. PayOrder в internal/order/payment.go
//     вызывает [Service.StartPayment]. В одной транзакции заказ становится paid,
//     создаётся попытка со своим saga_id и записывается команда DeductStock.
//     HTTP-ответ 202 означает, что попытка принята. Подтверждение клиент узнаёт
//     при чтении заказа по полю payment_confirmed.
//  2. Воркер outbox отправляет команду. Inventory списывает фильтр и отвечает
//     StockDeducted. [Service.HandleReply] завершает попытку и создаёт запись
//     для генерации справки. Если товара не хватило, StockRejected переводит
//     попытку в failed, а заказ обратно в active.
//  3. Если за 10 секунд ответ не успел примениться, [Service.RunDeadlines]
//     возвращает заказ в active и ставит в очередь CancelStock. Inventory
//     возвращает фильтр, если уже списал его по этой попытке.
//
// При чтении держите в голове три правила. saga_id обозначает попытку, order_id
// обозначает заказ. paid появляется до подтверждения списания. Ответ и дедлайн
// могут пересечься, поэтому право завершить попытку определяет условный UPDATE
// в PostgreSQL. Формат JSON, вызовы sqlc и частоту тика можно посмотреть позже.
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
