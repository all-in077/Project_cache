# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей
COPY Cache-service/go.mod Cache-service/go.sum ./
RUN go mod download

# Копируем исходный код
COPY Cache-service/ ./

# Собираем бинарник (CGO_ENABLED=0 нужен для статической линковки)
RUN CGO_ENABLED=0 GOOS=linux go build -o /cache-app cmd/main.go

# Stage 2: Final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем только бинарник из предыдущего этапа
COPY --from=builder /cache-app .

# Экспонируем порты (информативно)
EXPOSE 8080 50051

# Запуск приложения
CMD ["./cache-app"]