#!/bin/bash

# Скрипт для автоматизации деплоя Telegram-бота

set -e

echo "🚀 Начинаем процесс деплоя Telegram-бота..."

# Остановка текущего экземпляра бота
echo "⏹️ Останавливаем текущий экземпляр бота..."
make stop || true

# Обновление кода из репозитория
echo "⬇️ Обновляем код из репозитория..."
git pull

# Сборка Docker-образа
echo "🏗️ Собираем Docker-образ..."
make docker-build

# Остановка текущего Docker-контейнера
echo "⏹️ Останавливаем текущий Docker-контейнер..."
make docker-stop || true

# Запуск нового Docker-контейнера
echo "▶️ Запускаем новый Docker-контейнер..."
make docker-run

# Проверка статуса
echo "🔍 Проверяем статус..."
sleep 5
./scripts/docker-healthcheck.sh

echo "✅ Деплой успешно завершен!"
