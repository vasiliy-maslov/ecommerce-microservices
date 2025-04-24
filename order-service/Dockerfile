# Базовый образ для сборки
FROM golang:1.24-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod ./
COPY go.sum* ./
RUN go mod download

# Копируем весь код
COPY . .

# Компилируем приложение
RUN go build -o order-service ./cmd/order-service

# Финальный образ
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/order-service .
COPY --from=builder /app/migrations ./migrations/

# Устанавливаем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./order-service"]