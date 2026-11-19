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
// Ошибку чтения очереди или отправки Ack/Nack возвращает в runWorkers.
func consume(ctx context.Context, deliveries <-chan amqp.Delivery, handle Handler, logger *slog.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, open := <-deliveries:
			if !open {
				if ctx.Err() != nil {
					return nil
				}
				return errors.New("RabbitMQ deliveries closed")
			}
			if err := handleDelivery(ctx, delivery, handle, logger); err != nil {
				return err
			}
		}
	}
}

// handleDelivery вызывается из consume для одного сообщения.
//
// Сначала проверяет счётчик повторных доставок. При x-delivery-count >= 5
// пишет ошибку и подтверждает сообщение, чтобы убрать его из очереди.
// Иначе вызывает Handler. Успех даёт Ack, ошибка — Nack с возвратом в очередь
// после паузы. При остановке сервиса сообщение остаётся без подтверждения.
func handleDelivery(ctx context.Context, delivery amqp.Delivery, handle Handler, logger *slog.Logger) error {
	if count, _ := delivery.Headers["x-delivery-count"].(int64); count >= 5 {
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
			return delivery.Nack(false, true)
		}
	}
	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf("ack RabbitMQ delivery: %w", err)
	}
	return nil
}
