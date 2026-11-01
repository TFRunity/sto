module github.com/AlexeBabanov/sto/users

go 1.27

// go.work действует только локально; в Docker сборка идёт с GOWORK=off,
// поэтому chassis подключается через replace (ADR-036).
replace github.com/AlexeBabanov/sto/chassis => ../chassis

require (
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.12.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	golang.org/x/text v0.41.0 // indirect
)
