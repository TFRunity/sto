# Inventory (ADR-038, I-01). Контекст сборки — корень репозитория (ADR-036):
#   docker build -f inventory/Dockerfile .
# go.work в образ не копируется (GOWORK=off): chassis — через replace в go.mod.

FROM golang:1.27-alpine AS build
ENV CGO_ENABLED=0 GOWORK=off
WORKDIR /src

# Сначала только go.mod/go.sum — слой зависимостей кэшируется.
COPY chassis/go.mod chassis/go.sum* chassis/
COPY inventory/go.mod inventory/go.sum* inventory/
RUN cd inventory && go mod download

COPY chassis/ chassis/
COPY inventory/ inventory/
RUN cd inventory && go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/inventory

FROM alpine:3.23
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/app /usr/local/bin/app
USER app
EXPOSE 8080
# healthcheck (wget /healthz) — в docker-compose.yml
ENTRYPOINT ["/usr/local/bin/app"]
