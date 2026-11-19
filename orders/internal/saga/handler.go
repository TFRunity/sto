package saga

// HTTP-обработчик PayOrder находится в internal/order/payment.go рядом
// с другими операциями strict-сервера Orders. В этот пакет он входит
// через [Service.StartPayment].
