FROM golang:1.21-alpine AS builder

WORKDIR /workspace

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /workspace/geolocation .

FROM alpine:3.22.2

WORKDIR /app

COPY --from=builder /workspace/geolocation ./geolocation
COPY db/GeoLite2-City.mmdb ./db/GeoLite2-City.mmdb

ENV MAXMIND_DB_PATH=/app/db/GeoLite2-City.mmdb

EXPOSE 5000

ENTRYPOINT ["/app/geolocation"]
CMD ["runserver", "--http=5000"]
