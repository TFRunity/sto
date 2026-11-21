module github.com/AlexeBabanov/sto/services

go 1.27

// go.work действует только локально; в Docker сборка идёт с GOWORK=off,
// поэтому chassis подключается через replace (ADR-036).
replace github.com/AlexeBabanov/sto/chassis => ../chassis

require (
	github.com/AlexeBabanov/sto/chassis v0.0.0-00010101000000-000000000000
	github.com/getkin/kin-openapi v0.152.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.12.0
	github.com/oapi-codegen/runtime v1.8.0
	github.com/shopspring/decimal v1.5.0
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/caarlos0/env/v11 v11.4.1 // indirect
	github.com/go-openapi/jsonpointer v1.0.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx-shopspring-decimal v0.0.0-20220624020537-1d36b5a1853e // indirect
	github.com/jackc/puddle/v2 v2.2.3 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/oapi-codegen/nethttp-middleware v1.3.1 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/pressly/goose/v3 v3.29.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/sethvargo/go-retry v0.5.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)
