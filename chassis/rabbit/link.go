package rabbit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Route задаёт направления обмена для одного сервиса. Exchange и RoutingKey
// используются при отправке, Queue — при чтении. Очереди и привязки объявлены
// в rabbit_definitions в docker-compose.yml.
type Route struct {
	Exchange   string
	RoutingKey string
	Queue      string
}

// Run вызывается из outbox.RunRabbit, который запускают main.go Orders и Inventory.
//
// Порядок запуска:
//  1. openPublisher готовит канал отправки на первом соединении.
//  2. openDeliveries подписывается на очередь на втором соединении.
//  3. runWorkers запускает отправку из outbox и чтение очереди параллельно.
//
// При выходе из runWorkers оба цикла уже остановлены. Затем Run закрывает соединения.
func Run(ctx context.Context, brokerURL string, route Route, handle Handler, publish func(context.Context, *Publisher), logger *slog.Logger) error {
	publisherConn, err := amqp.Dial(brokerURL)
	if err != nil {
		return fmt.Errorf("connect RabbitMQ publisher: %w", err)
	}
	defer func() { _ = publisherConn.CloseDeadline(time.Now().Add(time.Second)) }()
	publisher, err := openPublisher(publisherConn, route)
	if err != nil {
		return fmt.Errorf("open RabbitMQ publisher: %w", err)
	}

	consumerConn, err := amqp.Dial(brokerURL)
	if err != nil {
		return fmt.Errorf("connect RabbitMQ consumer: %w", err)
	}
	defer func() { _ = consumerConn.CloseDeadline(time.Now().Add(time.Second)) }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	deliveries, err := openDeliveries(ctx, consumerConn, route.Queue)
	if err != nil {
		return fmt.Errorf("consume %s: %w", route.Queue, err)
	}
	return runWorkers(ctx, cancel, publisher, deliveries, handle, publish, logger)
}

// openPublisher вызывается из [Run] до запуска outbox. Он включает подтверждения
// брокера и регистрирует канал для возвратов. Оба сигнала читает [Publisher.Publish].
func openPublisher(conn *amqp.Connection, route Route) (*Publisher, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := channel.Confirm(false); err != nil {
		return nil, err
	}
	return &Publisher{
		conn: conn, channel: channel, returned: channel.NotifyReturn(make(chan amqp.Return, 1)),
		exchange: route.Exchange, key: route.RoutingKey,
	}, nil
}

// openDeliveries вызывается из [Run] после openPublisher. Проверяет, что очередь
// есть, ставит prefetch=1 и подписывается с ручным подтверждением. Канал доставок
// затем получает consume из consumer.go.
func openDeliveries(ctx context.Context, conn *amqp.Connection, queue string) (<-chan amqp.Delivery, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if _, err := channel.QueueDeclarePassive(queue, true, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := channel.Qos(1, 0, false); err != nil {
		return nil, err
	}
	return channel.ConsumeWithContext(ctx, queue, "", false, false, false, false, nil)
}

// runWorkers вызывается из [Run], когда оба канала готовы. publish запускает
// цикл outbox, а consume читает входящие сообщения.
//
// Если consume завершился или закрылся канал отправки, runWorkers отменяет
// общий контекст и дожидается обоих циклов. Ошибка уходит через Run и
// outbox.RunRabbit в группу сервиса, чтобы Compose мог его перезапустить.
func runWorkers(ctx context.Context, cancel context.CancelFunc, publisher *Publisher, deliveries <-chan amqp.Delivery, handle Handler, publish func(context.Context, *Publisher), logger *slog.Logger) error {
	closed := publisher.channel.NotifyClose(make(chan *amqp.Error, 1))
	published := make(chan struct{})
	go func() {
		defer close(published)
		publish(ctx, publisher)
	}()
	consumed := make(chan error, 1)
	go func() { consumed <- consume(ctx, deliveries, handle, logger) }()

	var err error
	select {
	case err = <-consumed:
		cancel()
	case <-closed:
		if ctx.Err() == nil {
			err = fmt.Errorf("RabbitMQ publisher channel closed")
		}
		cancel()
		<-consumed
	}
	<-published
	return err
}
