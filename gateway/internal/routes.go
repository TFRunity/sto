package internal

// TODO(dev): таблица «префикс → upstream + роли».
//   - deny-by-default: маршрута нет в таблице → 403/404               → ADR-019, P-01
//   - публичные маршруты недели 1: регистрация, вход (sequence-register,
//     sequence-engineer-login)
//   - роль не подходит → 403 конвертом                               → ADR-032
//   - префиксы согласуются с путями в api/openapi.yaml               → ADR-028
