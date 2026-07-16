FROM golang:1.26.5-alpine AS builder

RUN apk --no-cache add git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /out/geomesh \
    ./cmd/geomesh

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata bash curl && \
    addgroup -g 1001 -S geomesh && \
    adduser  -u 1001 -S geomesh -G geomesh

WORKDIR /app

COPY --from=builder /out/geomesh .
COPY scripts/download-geoip.sh ./scripts/download-geoip.sh

RUN chmod +x /app/geomesh /app/scripts/download-geoip.sh && \
    mkdir -p /app/config /app/geoip && \
    chown -R geomesh:geomesh /app

USER geomesh

EXPOSE 53/udp 53/tcp 8080/tcp

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD nc -zuv localhost 53 || exit 1

ENTRYPOINT ["/app/geomesh"]
CMD ["/app/config"]
