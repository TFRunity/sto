// Package stock — участник саги оплаты: списание и возврат остатков (ADR-030).
package stock

// TODO(dev): чистая функция «состояние попытки + команда → действие».
//   Состояния: нет попытки | deducted | rejected | cancelled.
//   Команды:   DeductStock | CancelStock.                            → ADR-030, D-02, D-07
//   - повторное DeductStock → тот же ответ, не молчаливый ack
//   - CancelStock без попытки → tombstone cancelled
//   - DeductStock по cancelled → отклонить
//   - табличный тест на ВСЮ таблицу состояние × команда              → ADR-013, D-07
