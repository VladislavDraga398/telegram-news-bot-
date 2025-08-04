package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/database"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/fetcher"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/utils"
)

// Scheduler управляет периодической отправкой новостей.
// Он будет запрашивать новости и рассылать их подписчикам.
type Scheduler struct {
	bot                 *tgbotapi.BotAPI
	userRepo            database.UserRepository
	subRepo             database.SubscriptionRepository
	sentArticleRepo     database.SentArticleRepository
	favoriteArticleRepo database.FavoriteArticleRepository
	fetcher             *fetcher.Fetcher
	interval            time.Duration
	stop                chan struct{}
	sentArticles        map[string]map[string]bool // Локальный кэш для оптимизации (будет постепенно заменен на БД)
}

// NewScheduler создает новый экземпляр планировщика.
func NewScheduler(
	bot *tgbotapi.BotAPI,
	userRepo database.UserRepository,
	subRepo database.SubscriptionRepository,
	sentArticleRepo database.SentArticleRepository,
	favoriteArticleRepo database.FavoriteArticleRepository,
	fetcher *fetcher.Fetcher,
	interval time.Duration,
) *Scheduler {
	return &Scheduler{
		bot:                 bot,
		userRepo:            userRepo,
		subRepo:             subRepo,
		sentArticleRepo:     sentArticleRepo,
		favoriteArticleRepo: favoriteArticleRepo,
		fetcher:             fetcher,
		interval:            interval,
		stop:                make(chan struct{}),
		sentArticles:        make(map[string]map[string]bool),
	}
}

// Start запускает цикл планировщика в отдельной горутине.
func (s *Scheduler) Start() {
	log.Println("Запуск планировщика новостей с интервалом:", s.interval)
	ticker := time.NewTicker(s.interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				s.sendNewsUpdates()
			case <-s.stop:
				ticker.Stop()
				log.Println("Планировщик новостей остановлен.")
				return
			}
		}
	}()
}

// Stop останавливает цикл планировщика.
func (s *Scheduler) Stop() {
	close(s.stop)
}

// IsArticleSent проверяет, была ли статья уже отправлена пользователю.
func (s *Scheduler) IsArticleSent(ctx context.Context, userID uint, articleURL string) (bool, error) {
	return s.isArticleSent(ctx, userID, articleURL), nil
}

// isArticleSent проверяет, была ли статья уже отправлена по данной теме.
func (s *Scheduler) isArticleSent(ctx context.Context, userID uint, articleURL string) bool {
	// Генерируем хеш статьи из URL
	articleHash := articleURL

	// Проверяем в базе данных, была ли статья отправлена
	sent, err := s.sentArticleRepo.IsArticleSent(ctx, userID, articleHash)
	if err != nil {
		log.Printf("Ошибка при проверке статьи в БД: %v", err)
		// В случае ошибки используем локальный кэш как запасной вариант
		topicKey := fmt.Sprintf("%d:%s", userID, articleHash)
		if _, ok := s.sentArticles[topicKey]; !ok {
			return false
		}
		return s.sentArticles[topicKey][articleURL]
	}

	return sent
}

// MarkArticleAsSent помечает статью как отправленную для данного пользователя.
func (s *Scheduler) MarkArticleAsSent(ctx context.Context, userID uint, articleURL string) error {
	// Помечаем в БД
	err := s.sentArticleRepo.MarkArticleAsSent(ctx, userID, articleURL)
	if err != nil {
		return err
	}

	// Помечаем в локальном кэше
	s.markArticleAsSent(ctx, userID, articleURL)
	return nil
}

// formatArticleMessage создает красиво отформатированное HTML-сообщение для новостной статьи
func (s *Scheduler) formatArticleMessage(article fetcher.Article) string {
	// Форматируем дату публикации
	publishedDate := article.PublishedAt.Format("02.01.2006 15:04")

	// Ограничиваем длину описания, чтобы избежать слишком длинных сообщений
	description := article.Description
	if len(description) > 300 {
		description = description[:297] + "..."
	}

	// Получаем название источника
	sourceName := article.Source.Name
	if sourceName == "" {
		sourceName = "Неизвестный источник"
	}

	// Создаем HTML-сообщение с форматированием
	message := fmt.Sprintf(
		"<b>%s</b>\n\n"+ // Заголовок жирным шрифтом
			"%s\n\n"+ // Описание
			"<i>📰 Источник: %s</i>\n"+ // Источник курсивом
			"<i>📅 Опубликовано: %s</i>\n\n"+ // Дата публикации курсивом
			"<a href=\"%s\">Читать полностью →</a>", // Ссылка на статью
		article.Title,
		description,
		sourceName,
		publishedDate,
		article.URL,
	)

	return message
}

// ResetSentArticlesHistory сбрасывает историю отправленных статей для указанного пользователя
func (s *Scheduler) ResetSentArticlesHistory(ctx context.Context, userID uint) error {
	// Сбрасываем историю в БД
	err := s.sentArticleRepo.ResetSentArticlesHistory(ctx, userID)
	if err != nil {
		return err
	}

	// Сбрасываем локальный кэш
	userIDStr := fmt.Sprintf("%d", userID)
	s.sentArticles[userIDStr] = make(map[string]bool)

	return nil
}

// markArticleAsSent помечает статью как отправленную для данного пользователя.
func (s *Scheduler) markArticleAsSent(ctx context.Context, userID uint, articleURL string) {
	// Генерируем хеш статьи из URL
	articleHash := articleURL

	// Сохраняем в базе данных
	err := s.sentArticleRepo.MarkArticleAsSent(ctx, userID, articleHash)
	if err != nil {
		log.Printf("Ошибка при сохранении статьи в БД: %v", err)
		// В случае ошибки используем локальный кэш как запасной вариант
		topicKey := fmt.Sprintf("%d:%s", userID, articleHash)
		if _, ok := s.sentArticles[topicKey]; !ok {
			s.sentArticles[topicKey] = make(map[string]bool)
		}

		// Ограничиваем размер кэша, чтобы он не рос бесконечно
		const maxCacheSize = 100
		if len(s.sentArticles[topicKey]) >= maxCacheSize {
			// Простой способ очистки: удаляем кэш и создаем заново.
			s.sentArticles[topicKey] = make(map[string]bool)
		}

		s.sentArticles[topicKey][articleURL] = true
	}
}

// sendNewsUpdates выполняет основную логику: получает темы, запрашивает новости и отправляет их.
func (s *Scheduler) sendNewsUpdates() {
	ctx := context.Background()
	log.Println("Планировщик: начинаю персональную проверку обновлений для пользователей...")

	users, err := s.userRepo.GetAllUsers(ctx)
	if err != nil {
		log.Printf("Планировщик: не удалось получить список пользователей: %v", err)
		return
	}

	log.Printf("Планировщик: найдено %d пользователей для проверки.", len(users))

	var wg sync.WaitGroup
	newsSentCount := 0
	mu := &sync.Mutex{}

	for _, user := range users {
		wg.Add(1)
		go func(u database.User) {
			defer wg.Done()
			foundNewsCount := s.ProcessUser(ctx, u, false) // Обычный запуск по расписанию
			mu.Lock()
			newsSentCount += foundNewsCount
			mu.Unlock()
		}(user)
	}
	wg.Wait()

	log.Println("Планировщик: проверка обновлений для всех пользователей завершена.")
}

// FetchNewsForTopic получает новости по конкретной теме.
func (s *Scheduler) FetchNewsForTopic(ctx context.Context, topic string) ([]fetcher.Article, error) {
	return s.fetcher.FetchNews(topic)
}

// SearchNews получает новости по произвольному поисковому запросу.
func (s *Scheduler) SearchNews(ctx context.Context, query string) ([]fetcher.Article, error) {
	// Используем тот же метод FetchNews, что и для поиска по теме
	return s.fetcher.FetchNews(query)
}

// AddFavoriteArticle добавляет статью в избранное пользователя.
func (s *Scheduler) AddFavoriteArticle(ctx context.Context, userID uint, article fetcher.Article) error {
	return s.favoriteArticleRepo.AddFavoriteArticle(
		ctx,
		userID,
		article.URL,
		article.Title,
		article.Source.Name,
		article.PublishedAt,
	)
}

// RemoveFavoriteArticle удаляет статью из избранного пользователя.
func (s *Scheduler) RemoveFavoriteArticle(ctx context.Context, userID uint, articleURL string) error {
	return s.favoriteArticleRepo.RemoveFavoriteArticle(ctx, userID, articleURL)
}

// GetUserFavoriteArticles возвращает список избранных статей пользователя.
func (s *Scheduler) GetUserFavoriteArticles(ctx context.Context, userID uint) ([]database.FavoriteArticle, error) {
	return s.favoriteArticleRepo.GetUserFavoriteArticles(ctx, userID)
}

// IsFavoriteArticle проверяет, добавлена ли статья в избранное пользователя.
func (s *Scheduler) IsFavoriteArticle(ctx context.Context, userID uint, articleURL string) (bool, error) {
	return s.favoriteArticleRepo.IsFavoriteArticle(ctx, userID, articleURL)
}

// sendArticleWithFavoriteButton отправляет новостную статью с кнопкой "В избранное"
func (s *Scheduler) sendArticleWithFavoriteButton(ctx context.Context, chatID int64, userID uint, article fetcher.Article) error {
	// Форматируем сообщение
	messageText := s.formatArticleMessage(article)

	// Проверяем, находится ли статья в избранном
	isFavorite, err := s.IsFavoriteArticle(ctx, userID, article.URL)
	if err != nil {
		log.Printf("Ошибка проверки избранной статьи: %v", err)
		// Продолжаем выполнение, даже если произошла ошибка
	}

	// Создаем короткий идентификатор для URL статьи
	shortID := utils.CreateShortID(article.URL)

	// Создаем клавиатуру с кнопкой "В избранное" или "Удалить из избранного"
	var keyboard tgbotapi.InlineKeyboardMarkup
	if isFavorite {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Удалить из избранного", "rm_fav_"+shortID),
			),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⭐ В избранное", "add_fav_"+shortID),
			),
		)
	}

	// Очищаем текст от некорректных символов
	sanitizedText := utils.SanitizeText(messageText)

	// Отправляем сообщение с клавиатурой
	msg := tgbotapi.NewMessage(chatID, sanitizedText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = false
	msg.ReplyMarkup = keyboard

	if _, err := s.bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки новости: %v", err)
		return err
	}

	return nil
}

// ProcessUser обрабатывает пользователя, отправляя ему новости по его подпискам.
// Возвращает количество отправленных новостей.
func (s *Scheduler) ProcessUser(ctx context.Context, user database.User, force bool) int {
	now := time.Now()
	interval := time.Duration(user.NotificationIntervalMinutes) * time.Minute

	// Проверяем, пора ли отправлять уведомление (если это не принудительный запуск)
	if !force && user.LastNotifiedAt != nil && now.Sub(*user.LastNotifiedAt) < interval {
		// Еще не время
		return 0
	}

	log.Printf("Планировщик: обрабатываю пользователя ID %d (TelegramID: %d)", user.ID, user.TelegramID)

	topics, err := s.subRepo.GetUserSubscriptions(ctx, user.ID)
	if err != nil {
		log.Printf("Планировщик: не удалось получить подписки для пользователя ID %d: %v", user.ID, err)
		return 0
	}

	if len(topics) == 0 {
		// У пользователя нет подписок, нечего отправлять
		return 0
	}

	var allFreshArticles []fetcher.Article
	newsFilterThreshold := time.Hour * 24 * 183 // 183 дня (примерно полгода)

	for _, topic := range topics {
		articles, err := s.fetcher.FetchNews(topic)
		if err != nil {
			log.Printf("Планировщик: ошибка при получении новостей для темы '%s': %v", topic, err)
			continue
		}

		for _, article := range articles {
			if now.Sub(article.PublishedAt) < newsFilterThreshold && !s.isArticleSent(ctx, user.ID, article.URL) {
				allFreshArticles = append(allFreshArticles, article)
				s.markArticleAsSent(ctx, user.ID, article.URL)
			}
		}
	}

	if len(allFreshArticles) == 0 {
		log.Printf("Планировщик: для пользователя ID %d новых статей не найдено.", user.ID)
		// Обновляем время, чтобы не проверять его снова на каждой итерации до истечения интервала
		if err := s.userRepo.UpdateUserLastNotifiedAt(ctx, user.ID, now); err != nil {
			log.Printf("Планировщик: не удалось обновить время последней проверки для пользователя ID %d: %v", user.ID, err)
		}
		return 0
	}

	// Ограничиваем количество новостей по настройкам пользователя
	newsLimit := int(user.NewsLimit)
	if newsLimit <= 0 {
		newsLimit = 5 // Значение по умолчанию, если вдруг в базе значение некорректное
	}

	// Отправляем новости с учетом ограничения
	articlesToSend := allFreshArticles
	if len(allFreshArticles) > newsLimit {
		articlesToSend = allFreshArticles[:newsLimit]
	}

	for _, article := range articlesToSend {
		// Используем метод sendArticleWithFavoriteButton для отправки новостей с кнопкой "В избранное"
		if err := s.sendArticleWithFavoriteButton(ctx, user.TelegramID, user.ID, article); err != nil {
			log.Printf("Планировщик: не удалось отправить новость пользователю ID %d: %v", user.ID, err)
			continue
		}
	}

	// Обновляем время последней отправки
	if err := s.userRepo.UpdateUserLastNotifiedAt(ctx, user.ID, now); err != nil {
		log.Printf("Планировщик: не удалось обновить время последней отправки для пользователя ID %d: %v", user.ID, err)
	}

	log.Printf("Планировщик: успешно отправлено %d новостей пользователю ID %d.", len(allFreshArticles), user.ID)

	return len(allFreshArticles)
}
