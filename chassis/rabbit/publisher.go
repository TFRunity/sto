// Package rabbit управляет отправкой и чтением сообщений RabbitMQ.
// Точка входа — [Run] в link.go.
package rabbit

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher создаётся в openPublisher и передаётся одному воркеру outbox.
type Publisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	returned <-chan amqp.Return
	exchange string
	key      string
}

// Publish вызывается из outbox.publishBatch по одному разу на строку outbox.
//
// Сначала отправляет сообщение с id строки. Затем ждёт подтверждение брокера
// и проверяет, не вернул ли брокер сообщение из-за отсутствующей привязки.
// Если Publish вернул nil, publishBatch помечает строку опубликованной.
// Подтверждение брокера касается отправки. Обработку на стороне получателя
// подтверждает отдельный Ack в handleDelivery (consumer.go).
func (p *Publisher) Publish(ctx context.Context, id, event string, body []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// amqp091-go v1.10 может ждать записи в сокет дольше контекста.
	// По таймауту закрываем соединение отправки. runWorkers заметит закрытие канала.
	closed := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(closed)
		_ = p.conn.CloseDeadline(time.Now())
	})
	defer func() {
		if !stop() {
			<-closed
		}
	}()

	return p.publishConfirmed(ctx, id, event, body)
}

func (p *Publisher) publishConfirmed(ctx context.Context, id, event string, body []byte) error {
	confirmation, err := p.channel.PublishWithDeferredConfirm(p.exchange, p.key, true, false, amqp.Publishing{
		MessageId: id, Type: event, ContentType: "application/json", DeliveryMode: amqp.Persistent, Body: body,
	})
	if err != nil {
		return err
	}
	ack, err := confirmation.WaitContext(ctx)
	if err != nil {
		return err
	}
	select {
	case returned, open := <-p.returned:
		if open {
			return fmt.Errorf("unroutable message: %s", returned.ReplyText)
		}
		return errors.New("publisher channel closed")
	default:
	}
	if !ack {
		return errors.New("broker rejected publication")
	}
	return nil
}
