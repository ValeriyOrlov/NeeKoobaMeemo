FROM golang:alpine AS builder

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код всего проекта
COPY . .

# Собираем сервис авторизации по относительному пути модуля
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./auth/cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/auth-service .
EXPOSE 8080
CMD ["./auth-service"]
