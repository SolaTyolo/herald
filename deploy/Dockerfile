FROM public.ecr.aws/docker/library/golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/herald-api ./cmd/api \
	&& CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/herald-worker ./cmd/worker

FROM public.ecr.aws/docker/library/alpine:3.21

RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=build /out/herald-api /out/herald-worker /app/
COPY config/providers.yaml /app/config/providers.yaml

ENV PROVIDERS_CONFIG=/app/config/providers.yaml
EXPOSE 8080
USER nobody
CMD ["/app/herald-api"]
