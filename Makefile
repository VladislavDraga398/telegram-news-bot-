## Makefile for Pet-Telegram-bot

.PHONY: run stop build clean test test-utils test-database test-handlers test-coverage lint docker-build docker-run docker-stop docker-push

# Имя бинарного файла, который будет создан
BINARY_NAME=telegram-bot
# Файл для хранения ID процесса (PID)
PID_FILE=.pidfile

# Команда для запуска
run: stop build
	@echo "Starting $(BINARY_NAME)..."
	# Запускаем бинарник в фоновом режиме, а его PID сохраняем в PID_FILE
	@./$(BINARY_NAME) & echo $$! > $(PID_FILE)
	@echo "Bot started with PID: $$(cat $(PID_FILE))"

# Команда для остановки
stop:
	@echo "Stopping bot..."
	@if [ -f $(PID_FILE) ]; then \
		kill $$(cat $(PID_FILE)) || true; \
		rm -f $(PID_FILE); \
		echo "Bot stopped."; \
	else \
		echo "Bot is not running or PID file not found."; \
	fi

# Команда для сборки
build:
	@echo "Building binary..."
	@go build -o $(BINARY_NAME) ./cmd/bot
	@echo "Binary built successfully."

# Команда для очистки
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME) $(PID_FILE)
	@echo "Cleanup complete."

# Команды для тестирования
test:
	@echo "Running all tests..."
	@go test ./tests/...
	@echo "All tests completed."

test-utils:
	@echo "Running utils tests..."
	@go test ./tests/utils/

test-database:
	@echo "Running database tests..."
	@go test ./tests/database/

test-handlers:
	@echo "Running handlers tests..."
	@go test ./tests/handlers/

test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./tests/...
	@echo "Coverage report completed."

# Команда для линтинга
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Using from GOPATH..."; \
		$(shell go env GOPATH)/bin/golangci-lint run || { \
			echo "golangci-lint not found in GOPATH. Installing..."; \
			go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
			$(shell go env GOPATH)/bin/golangci-lint run; \
		}; \
	fi
	@echo "Linting completed."

# Docker команды
docker-build:
	@echo "Building Docker image..."
	docker build -t telegram-news-bot:latest .
	@echo "Docker image built successfully."

docker-run: docker-build
	@echo "Starting Docker container..."
	docker-compose up -d
	@echo "Docker container started."

docker-stop:
	@echo "Stopping Docker container..."
	docker-compose down
	@echo "Docker container stopped."

docker-push: docker-build
	@echo "Pushing Docker image to registry..."
	@echo "Please login to your Docker registry first using 'docker login'"
	@read -p "Enter Docker registry username: " USERNAME; \
	docker tag telegram-news-bot:latest $${USERNAME}/telegram-news-bot:latest; \
	docker push $${USERNAME}/telegram-news-bot:latest
	@echo "Docker image pushed successfully."
