package saga

// TODO(dev): HTTP оплаты (strict-сервер oapi-codegen).               → ADR-028
//   - оплата клиентом → 202 Accepted; клиент перечитывает заказ      → ADR-030 (А-06)
//   - заказ не active → 409 конвертом                                → ADR-032
//   - без автоматического повтора                                    → ADR-016
