package certificate

// TODO(dev): репозиторий справок.
//   - выбор pending | failed с next_attempt_at <= now()               → ADR-035
//   - переходы статуса — условный UPDATE                             → ADR-031
//   - pdf + snapshot сохраняются при completed
//   - чтение PDF для скачивания                                      → sequence-download-certificate
// Создание справки — в репозитории саги, не здесь (ADR-035).
