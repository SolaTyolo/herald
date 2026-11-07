# All-in-one: herald-api + N× herald-worker (DELIVERY_WORKER_COUNT).
# Remote Delivery Workers are separate images — not bundled here.
# Publish only API :8080.
FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/herald-api ./cmd/api \
	&& CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/herald-worker ./cmd/worker

FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=build /out/herald-api /out/herald-worker /app/
COPY config/providers.yaml /app/config/providers.yaml
COPY deploy/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

ENV PROVIDERS_CONFIG=/app/config/providers.yaml
ENV DELIVERY_WORKER_COUNT=1
EXPOSE 8080
USER nobody
ENTRYPOINT ["/app/entrypoint.sh"]
