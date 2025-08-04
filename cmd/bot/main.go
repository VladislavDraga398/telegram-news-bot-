package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/config"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/database"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/fetcher"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/handlers"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/scheduler"
)

func main() {
	// 1. Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// 2. Инициализация базы данных
	dbConn, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	db := dbConn.GetDB() // Получаем *gorm.DB из интерфейса

	// Запускаем миграцию для исправления регистра старых подписок
	if err := database.MigrateSubscriptionsToLower(db); err != nil {
		log.Printf("Ошибка миграции данных: %v", err)
	}

	// 3. Инициализация бота
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		log.Fatalf("Ошибка инициализации бота: %v", err)
	}
	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	// 4. Создание репозиториев
	userRepo := database.NewUserRepository(db)
	subRepo := database.NewSubscriptionRepository(db)
	sentArticleRepo := database.NewSentArticleRepository(db)
	favoriteArticleRepo := database.NewFavoriteArticleRepository(db)

	// 5. Инициализация Fetcher и Scheduler
	// Передаем оба API ключа
	newsFetcher := fetcher.NewFetcher(cfg.GNewsAPIKey, cfg.NewsAPIKey)
	// Интервал проверки - 1 минута (для теста)
	newsScheduler := scheduler.NewScheduler(bot, userRepo, subRepo, sentArticleRepo, favoriteArticleRepo, newsFetcher, 1*time.Minute)

	// 6. Создание обработчика
	handler := handlers.NewHandler(bot, userRepo, subRepo, newsScheduler)

	// 7. Настройка и запуск
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if cfg.Mode == "webhook" {
		log.Fatal("Режим Webhook пока не поддерживается в этой конфигурации.")
		// TODO: Добавить graceful shutdown и для webhook
	} else {
		log.Println("Бот запущен в режиме long polling")

		// Запускаем планировщик
		newsScheduler.Start()

		// Настраиваем канал для получения обновлений.
		updates := bot.GetUpdatesChan(tgbotapi.UpdateConfig{
			Offset:  0,
			Timeout: 60,
		})

		// Запускаем обработку обновлений в отдельной горутине.
		go func() {
			for update := range updates {
				go handler.HandleUpdate(update)
			}
		}()

		// Ожидаем сигнал для завершения работы.
		<-sigChan
		log.Println("Получен сигнал завершения, останавливаем сервисы...")

		// Останавливаем планировщик
		newsScheduler.Stop()

		// Аккуратно останавливаем получение новых сообщений.
		bot.StopReceivingUpdates()
		log.Println("Бот успешно остановлен.")
	}
}
