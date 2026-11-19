package stock

import "github.com/AlexeBabanov/sto/chassis/stockmessage"

// Эти состояния сохраняются в stock_attempts для конкретного saga_id.
const (
	deducted  = "deducted"
	rejected  = "rejected"
	cancelled = "cancelled"
)

// action — результат decide. Его выполняет applyCommand в repository.go.
type action int

const (
	ignore         action = iota // Отмена уже обработана или списание ранее отклонено.
	deduct                       // Впервые попробовать списать все позиции команды.
	repeatDeducted               // Снова отправить StockDeducted, сохранив текущий остаток.
	repeatRejected               // Ответить StockRejected, сохранив текущий остаток.
	restore                      // Вернуть количества из stock_attempt_items.
	tombstone                    // Отмена пришла первой. lockAttempt уже записал cancelled.
)

// decide выбирает действие по состоянию попытки и типу команды.
//
// Порядок вызовов:
//  1. repository.handle в consumer.go разбирает сообщение и вызывает repository.apply.
//  2. Внутри транзакции lockAttempt резервирует новую попытку или читает существующую.
//     Пустая строка означает, что попытку только что создали этой командой.
//  3. decide возвращает действие. applyCommand исполняет его в той же транзакции.
//
// Повтор DeductStock требует ответа даже при уже известном результате: предыдущий
// ответ мог потеряться. После CancelStock поздний DeductStock получает отказ,
// чтобы товар не списался по уже отменённой попытке.
func decide(state, event string) action {
	switch event {
	case stockmessage.DeductStock:
		switch state {
		case "":
			return deduct
		case deducted:
			return repeatDeducted
		case rejected, cancelled:
			return repeatRejected
		}
	case stockmessage.CancelStock:
		switch state {
		case "":
			return tombstone
		case deducted:
			return restore
		}
	}
	return ignore
}
