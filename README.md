# 🤖 InfoRaptor - Telegram News Bot

<div align="center">

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-26A5E4?style=for-the-badge&logo=telegram&logoColor=white)
![CI/CD](https://github.com/VladislavDraga398/telegram-news-bot-/actions/workflows/ci.yml/badge.svg)

**Умный Telegram-бот для отслеживания новостей по ключевым словам**

[Особенности](#-особенности) • [Технологии](#-технологии) • [Установка](#-установка) • [Использование](#-использование) • [Архитектура](#-архитектура)

</div>

---

## 📋 О проекте

InfoRaptor — это интеллектуальный Telegram-бот, разработанный для автоматического отслеживания новостей по заданным темам. Проект создан как демонстрация навыков разработки на Go и работы с современным стеком технологий.

### 🎯 Цели проекта
- Демонстрация навыков программирования на Go
- Работа с Telegram Bot API
- Проектирование архитектуры микросервисов
- Использование современных инструментов DevOps
- Создание качественного портфолио проекта

## ✨ Особенности

### 🔍 Управление новостями
- **Поиск новостей** - Поиск статей по ключевым словам
- **Избранное** - Сохранение и управление понравившимися статьями
- **Последние новости** - Получение актуальных новостных сводок

### 📬 Система подписок
- **Подписка на темы** - Автоматическое отслеживание интересующих тем
- **Управление подписками** - Легкое добавление и удаление тем
- **Персонализация** - Настройка частоты уведомлений

### 🎨 Пользовательский интерфейс
- **Интуитивные клавиатуры** - Быстрое взаимодействие через inline-кнопки
- **Красивое оформление** - Эмодзи и форматированные сообщения
- **Многоязычность** - Поддержка русского языка и эмодзи

### 🛡️ Надежность
- **Graceful Shutdown** - Корректное завершение работы
- **Обработка ошибок** - Устойчивость к сбоям
- **Логирование** - Подробное отслеживание работы системы

## 🛠 Технологии

### Backend
- **[Go 1.23+](https://golang.org/)** - Основной язык программирования
- **[GORM](https://gorm.io/)** - ORM для работы с базой данных
- **[SQLite](https://www.sqlite.org/)** - Легковесная база данных (CGO-free драйвер)
- **[Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api)** - Взаимодействие с Telegram

### DevOps & Инфраструктура
- **[Docker](https://www.docker.com/)** - Контейнеризация приложения
- **[Docker Compose](https://docs.docker.com/compose/)** - Оркестрация контейнеров
- **[GitHub Actions](https://github.com/features/actions)** - CI/CD пайплайны
- **[golangci-lint](https://golangci-lint.run/)** - Статический анализ кода

### Инструменты разработки
- **[godotenv](https://github.com/joho/godotenv)** - Управление переменными окружения
- **[Makefile](https://www.gnu.org/software/make/)** - Автоматизация задач сборки
- **Go modules** - Управление зависимостями

## 🚀 Установка

### Предварительные требования
- Go 1.23 или выше
- Docker и Docker Compose (для контейнерной установки)
- Токен Telegram Bot (получить у [@BotFather](https://t.me/botfather))

### 📦 Быстрый старт с Docker

1. **Клонирование репозитория**
```bash
git clone https://github.com/VladislavDraga398/telegram-news-bot.git
cd telegram-news-bot
```

2. **Настройка окружения**
```bash
cp .env.example .env
# Отредактируйте .env файл, добавив ваш TELEGRAM_BOT_TOKEN
```

3. **Запуск с Docker**

```bash
# Использование готового образа из GitHub Container Registry
docker pull ghcr.io/vladislavdraga398/telegram-news-bot:latest
docker run -d --name telegram-news-bot -e TELEGRAM_BOT_TOKEN=your_token ghcr.io/vladislavdraga398/telegram-news-bot:latest
```

4. **Сборка и запуск с Docker Compose**
```bash
make docker-run
```

4. **Остановка**
```bash
make docker-stop
```

### 🔧 Локальная разработка

1. **Установка зависимостей**
```bash
go mod download
```

2. **Настройка базы данных**
```bash
mkdir -p data
```

3. **Запуск в режиме разработки**
```bash
make run
```

4. **Остановка**
```bash
make stop
```

## ⚙️ Конфигурация

Создайте файл `.env` в корне проекта:

```env
# Обязательные параметры
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here

# Опциональные параметры
DB_PATH=./data/news_bot.db
LOG_LEVEL=info
NEWS_CHECK_INTERVAL=1m
MAX_NEWS_PER_REQUEST=5
```

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `TELEGRAM_BOT_TOKEN` | Токен Telegram бота | **Обязательно** |
| `DB_PATH` | Путь к файлу базы данных | `./data/news_bot.db` |
| `LOG_LEVEL` | Уровень логирования | `info` |
| `NEWS_CHECK_INTERVAL` | Интервал проверки новостей | `1m` |
| `MAX_NEWS_PER_REQUEST` | Максимум новостей за запрос | `5` |

## 📱 Использование

### Основные команды

- `/start` - Начать работу с ботом
- `/help` - Получить справку по командам
- `/subscribe <тема>` - Подписаться на тему
- `/unsubscribe <тема>` - Отписаться от темы
- `/subscriptions` - Показать все подписки
- `/search <запрос>` - Поиск новостей
- `/favorites` - Управление избранными статьями
- `/latest` - Последние новости

### Примеры использования

```
/subscribe технологии
/search искусственный интеллект
/latest
```

## 🏗 Архитектура

```
telegram-news-bot/
├── cmd/bot/                 # Точка входа приложения
│   └── main.go
├── internal/bot/            # Основная бизнес-логика
│   ├── config/             # Конфигурация
│   ├── database/           # Работа с БД и модели
│   ├── fetcher/            # Получение новостей
│   ├── handlers/           # Обработчики команд
│   ├── scheduler/          # Планировщик задач
│   ├── server/             # HTTP сервер (health checks)
│   └── utils/              # Утилитарные функции
├── tests/                   # Тесты
│   ├── database/           # Тесты БД
│   ├── handlers/           # Тесты обработчиков
│   └── utils/              # Тесты утилит
├── scripts/                # Скрипты развертывания
├── .github/workflows/      # CI/CD пайплайны
├── Dockerfile              # Docker образ
├── docker-compose.yml      # Docker Compose конфигурация
├── Makefile               # Автоматизация задач
└── README.md              # Документация
```

### Основные компоненты

- **Handlers** - Обработка команд и callback'ов от пользователей
- **Database** - Слой работы с данными (пользователи, подписки, избранное)
- **Fetcher** - Получение новостей из внешних источников
- **Scheduler** - Периодическая отправка новостей подписчикам
- **Utils** - Вспомогательные функции (санитизация текста, создание ID)

## 🧪 Тестирование

### Запуск всех тестов
```bash
make test
```

### Запуск тестов с покрытием
```bash
make test-coverage
```

### Запуск тестов по компонентам
```bash
make test-database    # Тесты базы данных
make test-handlers    # Тесты обработчиков
make test-utils       # Тесты утилит
```

## 🔧 Разработка

### Доступные команды Make

```bash
make build           # Сборка приложения
make run             # Запуск в режиме разработки
make stop            # Остановка приложения
make test            # Запуск тестов
make test-coverage   # Тесты с покрытием
make lint            # Статический анализ кода
make docker-build    # Сборка Docker образа
make docker-run      # Запуск в Docker
make docker-stop     # Остановка Docker контейнера
make clean           # Очистка артефактов сборки
```

### Стандарты кода

- Следование [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Использование `golangci-lint` для статического анализа
- Покрытие тестами критических компонентов
- Документирование публичных функций и методов

## 🐳 Docker

### Особенности Docker образа

- **Многоэтапная сборка** - Минимальный размер финального образа
- **CGO-free сборка** - Статический бинарник без внешних зависимостей
- **Безопасность** - Запуск от непривилегированного пользователя
- **Health checks** - Мониторинг состояния контейнера

### Переменные окружения в Docker

```yaml
environment:
  - TZ=Europe/Moscow
  - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
  - DB_PATH=/app/data/news_bot.db
```

## 📊 Мониторинг

### Health Check

Бот предоставляет HTTP endpoint для проверки состояния:

```bash
curl http://localhost:8080/health
```

### Логирование

- Структурированные логи в JSON формате
- Различные уровни логирования (debug, info, warn, error)
- Ротация логов в production окружении

## 🤝 Вклад в проект

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'feat: Добавление новой функции'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

Этот проект распространяется под лицензией MIT. Подробности в файле [LICENSE](LICENSE).

## 👨‍💻 Автор

**Владислав Драгоненков**
- GitHub: [@VladislavDraga398](https://github.com/VladislavDraga398)
- Telegram: [@Vladislav_Dragonenkov](https://t.me/Vladislav_Dragonenkov)
- Email: dragonencov@gmail.com

---

<div align="center">

**⭐ Если проект понравился, поставьте звездочку! ⭐**

Made with ❤️ in Russia

</div>
