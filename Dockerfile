FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN go build -trimpath -ldflags="-s -w" -o /out/fireproxy ./cmd/fireproxy

FROM alpine:3.21

RUN adduser -D -u 10001 app
USER app
WORKDIR /app

COPY --from=build /out/fireproxy /usr/local/bin/fireproxy

EXPOSE 8080

ENTRYPOINT ["fireproxy"]
