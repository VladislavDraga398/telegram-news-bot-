FROM golang:1.23-alpine

# Установка необходимых пакетов
RUN apk add --no-cache git make tzdata ca-certificates

# Установка рабочей директории
WORKDIR /app

# Копирование файлов go.mod и go.sum
COPY go.mod go.sum ./

# Очистка кэша модулей и загрузка зависимостей
RUN go clean -modcache && go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения без CGO, так как используем альтернативный драйвер SQLite
RUN CGO_ENABLED=0 GOOS=linux go build -tags="!cgo" -ldflags="-s -w" -o telegram-bot ./cmd/bot

# Создание директории для данных
RUN mkdir -p /app/data

# Установка переменной окружения для пути к базе данных
ENV DB_PATH=/app/data/news_bot.db

# Проверка работоспособности
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 CMD ps aux | grep telegram-bot || exit 1

# Запуск приложения
CMD ["/app/telegram-bot"]
