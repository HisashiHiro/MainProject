# ---- Стадия сборки ----
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Копируем файлы модулей и загружаем зависимости
COPY go.mod go.sum ./
RUN go mod download -x

# Копируем весь исходный код
COPY . .

# Собираем статически скомпилированный бинарник
RUN go build -o /app/notes-app ./cmd/NotesApp/main.go

# ---- Финальный образ ----
FROM alpine:latest

# Устанавливаем корневые сертификаты (для HTTPS-вызовов)
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Копируем бинарник из стадии сборки
COPY --from=builder /app/notes-app /app/notes-app

# Копируем папки с миграциями и шаблонами (необходимы для работы приложения)
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/templates /app/templates

# Открываем порты HTTP и gRPC
EXPOSE 8080 50051

# Запускаем приложение
CMD ["/app/notes-app"]