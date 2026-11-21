# v2 — отложенные модули

Что здесь:

| Путь | Содержимое |
|---|---|
| `adr/architecture/ADR-018-chat-websocket-exception.md` | WebSocket для чата |
| `uml/sequence/` | `sequence-chat-client/engineer/director`, `sequence-report`, `sequence-employee-stats` |
| `uml/use-case/use-case-diagram-v4-full.drawio` | **полная** use case diagram (v1 + v2) |
| `uml/class/class-diagram-full.drawio` | **полная** диаграмма классов (v1 + v2) |
| `design/web-director/` | экраны директора 3, 3b, 4, 7, 9 (сотрудники, отчёты, чат) |
| `design/web-general/` | навигация «Отчёты открыт», чат клиента |
| `design/react-native/` | вкладка и экраны «Чат», push с чат-уведомлением |
| `TODO.md` | открытые вопросы по Chat / Reports |

Полные диаграммы (`*-full.drawio`) — источник для слияния: в v1 лежат их урезанные копии.

## Чек-лист возврата v2 в v1

1. Положить `sequence-*` из `uml/sequence/` в `v1/uml/sequence/`; заменить use case и class diagram
   полными версиями (или повторно применить правки).
2. Вернуть `ADR-018` в `v1/adr/architecture/`; обновить `v1/adr/README.md`.
3. ADR-020: убрать пометки «(в v2 …)» у диалога в чате и отчёта «по сотрудникам».
4. Дизайн: вернуть пункты навигации («Отчёты», «Чат» у директора, «Чаты» у клиента), вкладку
   «Чат» в RN, экраны 3 / 3b / 4 / 7 / 9 (директор), чат клиента и экраны 6 / 6b (RN),
   чат-уведомление в push; в HTML — `*-v2.html` склеить с v1.
5. Инфраструктура: сервисы `chat` и `reports` и их БД в `docker-compose.yml`; маршруты Chat
   (HTTP + WebSocket) и Reports в Gateway; очереди Reports (`reports.data`), потребители команд
   отчёта в Orders и Users.
6. «Поддержка» (RN-профиль, сообщение после 3 неудач справки): статические контакты → диалог
   `support` в Chat.
7. Дашборд директора: решить, остаётся ли он на Orders / Users или метрики переходят в Reports.
8. CI (ADR-014): добавить новые сервисы в сборку.
