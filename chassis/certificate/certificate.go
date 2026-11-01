// Package certificate — HTTP-адаптер к Gotenberg: шаблон + данные → PDF (ADR-027).
package certificate

// TODO(dev): генератор PDF.
//   - POST /forms/chromium/convert/html, multipart, поле files,
//     главный файл — index.html                                     → ADR-027, infra «Сверка»
//   - TTF с кириллицей через go:embed, в том же запросе; @font-face по
//     имени файла, без подкаталогов
//   - html/template парсится один раз при старте (go:embed)
//   - http.Client с таймаутом; сбой → попытка неудачна (повторы — ADR-035)
// Сигнатура интерфейса генератора не решена (TODO.md §5) — за тобой.
