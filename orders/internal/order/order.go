// Package order — заказ: создание, позиции, чтение (ADR-021).
package order

// TODO(dev): доменные типы и чистые правила.                          → D-07, ADR-031
//   - статусы requested → active → paid / cancelled                  → ADR-021
//   - сумма заказа по сохранённым ценам позиций, decimal             → ADR-034, ADR-026
//   - canStartPayment: только из active
//   - табличные тесты на сумму и canStartPayment                     → ADR-013
