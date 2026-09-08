# ЭТАП 1: Сборка игрового бэкенда
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o game-service cmd/server/main.go

# ЭТАП 2: Финальный образ с веб-ресурсами
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/game-service .

# Копирование статических ресурсов фронтенда
COPY --from=builder /app/web ./web
COPY --from=builder /app/pictures ./pictures
COPY --from=builder /app/sounds ./sounds

EXPOSE 8081

CMD ["./game-service"]