FROM golang:alpine AS builder

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код всего проекта
COPY . .

# Собираем игровой сервис по относительному пути модуля
RUN CGO_ENABLED=0 GOOS=linux go build -o game-service ./game/cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /app/game-service .
COPY --from=builder /app/game/web ./web
COPY --from=builder /app/game/pictures ./pictures
COPY --from=builder /app/game/sounds ./sounds

EXPOSE 8081
CMD ["./game-service"]
