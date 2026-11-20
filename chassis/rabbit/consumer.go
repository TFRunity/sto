package rabbit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Handler — обработчик сервиса. Orders передаёт saga.Service.HandleReply через
// замыкание, Inventory — результат stock.NewHandler. Для успешной обработки
// Handler возвращает nil после коммита, чтобы handleDelivery мог отправить Ack.
type Handler func(ctx context.Context, event string, body []byte) error

// consume запускается из runWorkers (link.go). Он читает по одной доставке
// из канала, открытого openDeliveries, и передаёт её в handleDelivery.
// Ошибку чтения очереди или отправки Ack/Reject возвращает в runWorkers.
func consume(ctx context.Context, deliveries <-chan amqp.Delivery, handle Handler, logger *slog.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, open := <-deliveries:
			if !open {
				select {
				case <-ctx.Done():
					return nil
				default:
					return errors.New("RabbitMQ deliveries closed")
				}
			}
			if err := handleDelivery(ctx, delivery, handle, logger); err != nil {
				return err
			}
		}
	}
}

// handleDelivery вызывается из consume для одного сообщения.
//
// Сначала проверяет счётчик неудачных доставок. После четырёх неудач
// пишет ошибку и подтверждает пятую доставку, чтобы убрать сообщение из очереди.
// Иначе вызывает Handler. Успех даёт Ack, ошибка — Reject с возвратом в очередь
// после паузы. При остановке сервиса сообщение остаётся без подтверждения.
func handleDelivery(ctx context.Context, delivery amqp.Delivery, handle Handler, logger *slog.Logger) error {
	const maxFailedDeliveries = 4
	if count, _ := delivery.Headers["x-delivery-count"].(int64); count >= maxFailedDeliveries {
		logger.ErrorContext(ctx, "discarding message after retries", "id", delivery.MessageId, "event", delivery.Type)
		return delivery.Ack(false)
	}
	if err := handle(ctx, delivery.Type, delivery.Body); err != nil {
		logger.WarnContext(ctx, "message handling failed", "id", delivery.MessageId, "err", err)
		timer := time.NewTimer(200 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil // RabbitMQ вернёт сообщение в очередь при закрытии соединения.
		case <-timer.C:
			// RabbitMQ 4.3 засчитывает Reject с возвратом как неудачную доставку.
			return delivery.Reject(true)
		}
	}
	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf("ack RabbitMQ delivery: %w", err)
	}
	return nil
}
