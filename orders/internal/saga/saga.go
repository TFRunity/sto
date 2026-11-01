// Package saga — оркестратор саги оплаты (ADR-003, ADR-030).
package saga

// TODO(dev): чистая функция шага саги.                               → D-07, ADR-031
//   - «current_step + ответ → завершить / откатить / проигнорировать»
//   - шаги: verifying_stock → completed | failed                    → ADR-030
//   - ответ на неизвестную сагу или пройденный шаг → игнор (ack + warn)
//   - табличный тест                                                 → ADR-013
