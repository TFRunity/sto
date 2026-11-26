FROM golang:1.27-alpine AS build
ENV CGO_ENABLED=0 GOWORK=off
WORKDIR /src

COPY chassis/go.mod chassis/go.sum chassis/
COPY chat/go.mod chat/go.sum chat/
RUN cd chat && go mod download

COPY chassis/ chassis/
COPY chat/ chat/
RUN cd chat && go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/chat

FROM alpine:3.23
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/app /usr/local/bin/app
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/app"]
