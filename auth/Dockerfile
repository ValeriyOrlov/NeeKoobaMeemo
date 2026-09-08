# ЭТАП 1: Сборка бинарника Go
FROM golang:alpine AS builder

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем легкий статически скомпилированный бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service cmd/server/main.go

# ЭТАП 2: Минимальный образ для запуска
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/auth-service .

EXPOSE 8080

CMD ["./auth-service"]