# Stage 1: Build stage
FROM golang:1.23-alpine AS builder

# Установим рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Устанавливаем зависимости
RUN go mod download

# Копируем исходный код приложения
COPY . .

# Копируем папку docs (удалите эту строку, если вы используете COPY . . для всего проекта)
COPY ./docs /app/docs

# Компилируем приложение
RUN GOOS=linux GOARCH=amd64 go build -o tg-dice-backend ./cmd/main.go

# Stage 2: Runtime stage
FROM alpine:latest

# Устанавливаем необходимые пакеты (например, для работы с сертификатами)
RUN apk --no-cache add ca-certificates

# Копируем исполняемый файл и документацию
COPY --from=builder /app/tg-dice-backend /usr/local/bin/tg-dice-backend
COPY --from=builder /app/docs /app/docs

# Настраиваем переменные окружения
ENV PORT=8080 \
    MONGO_URI=mongodb://root:example@mongo:27017/tgdice?authSource=admin \
    DB_NAME=tgdice \
    REDIS_HOST=redis \
    REDIS_PORT=6379 \
    REDIS_PASSWORD=yourpassword

# Открываем порт
EXPOSE 8080

# Указываем команду для запуска
CMD ["tg-dice-backend"]
