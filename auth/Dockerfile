# Этап сборки
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей сначала (для лучшего кэширования)
COPY go.mod go.sum ./
RUN go mod download

# Копируем и собираем исходный код
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/app/main.go

# Финальный этап - минимальный образ
FROM alpine:3.19

# Устанавливаем зависимости времени выполнения если нужны
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Копируем бинарник из этапа сборки
COPY --from=builder /app/main /app/main

COPY .env .env

# Создаем непривилегированного пользователя для безопасности
RUN adduser -D -g '' appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8012

CMD ["/app/main"]