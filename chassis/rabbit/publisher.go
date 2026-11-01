// Package rabbit — обёртки над amqp091-go: публикация и потребление (ADR-025).
package rabbit

// TODO(dev): публикатор.
//   - отдельное соединение для публикации; канал не потокобезопасен    → ADR-025
//   - publisher confirms; ack брокера → outbox помечает published      → ADR-025, ADR-005
//   - exchange и routing key — из definitions в docker-compose.yml
//     (routing key = имя очереди)
