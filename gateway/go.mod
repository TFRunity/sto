module github.com/AlexeBabanov/sto/gateway

go 1.27

// go.work действует только локально; в Docker сборка идёт с GOWORK=off,
// поэтому chassis подключается через replace (ADR-036).
replace github.com/AlexeBabanov/sto/chassis => ../chassis

require (
	github.com/AlexeBabanov/sto/chassis v0.0.0-00010101000000-000000000000
	github.com/getkin/kin-openapi v0.152.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/oapi-codegen/runtime v1.8.0
	github.com/shopspring/decimal v1.5.0
	golang.org/x/sync v0.24.0
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/caarlos0/env/v11 v11.4.1 // indirect
	github.com/go-openapi/jsonpointer v1.0.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/oapi-codegen/nethttp-middleware v1.3.1 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	golang.org/x/text v0.43.0 // indirect
)
