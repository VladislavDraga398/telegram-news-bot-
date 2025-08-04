#!/bin/bash

# Скрипт для проверки работоспособности Docker-контейнера

echo "Проверка Docker-контейнера telegram-news-bot..."

# Проверяем, запущен ли контейнер
CONTAINER_ID=$(docker ps -q -f name=telegram-news-bot)

if [ -z "$CONTAINER_ID" ]; then
    echo "❌ Контейнер не запущен"
    exit 1
else
    echo "✅ Контейнер запущен с ID: $CONTAINER_ID"
fi

# Проверяем логи контейнера
echo "Последние логи контейнера:"
docker logs --tail 10 telegram-news-bot

# Проверяем статус контейнера
HEALTH_STATUS=$(docker inspect --format='{{.State.Health.Status}}' telegram-news-bot 2>/dev/null)

if [ -z "$HEALTH_STATUS" ]; then
    echo "ℹ️ Контейнер не имеет настроенного healthcheck"
else
    echo "Статус здоровья: $HEALTH_STATUS"
fi

echo "Проверка завершена"
