package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/database"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/fetcher"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/utils"
)

const (
	StateDefault             = ""
	StateAwaitingTopic       = "awaiting_topic"
	StateAwaitingSearchQuery = "awaiting_search_query"
	StateViewingFavorites    = "viewing_favorites"
)

// Scheduler is an interface that the scheduler must implement.
// This avoids a circular dependency.
type Scheduler interface {
	ProcessUser(ctx context.Context, user database.User, force bool) int
	FetchNewsForTopic(ctx context.Context, topic string) ([]fetcher.Article, error)
	SearchNews(ctx context.Context, query string) ([]fetcher.Article, error)
	IsArticleSent(ctx context.Context, userID uint, articleURL string) (bool, error)
	MarkArticleAsSent(ctx context.Context, userID uint, articleURL string) error
	ResetSentArticlesHistory(ctx context.Context, userID uint) error
	AddFavoriteArticle(ctx context.Context, userID uint, article fetcher.Article) error
	RemoveFavoriteArticle(ctx context.Context, userID uint, articleURL string) error
	GetUserFavoriteArticles(ctx context.Context, userID uint) ([]database.FavoriteArticle, error)
	IsFavoriteArticle(ctx context.Context, userID uint, articleURL string) (bool, error)
}

// Handler processes incoming updates from Telegram
// and manages the bot's state.
type Handler struct {
	bot       *tgbotapi.BotAPI
	userRepo  database.UserRepository
	subRepo   database.SubscriptionRepository
	scheduler Scheduler
}

// NewHandler creates a new handler instance.
func NewHandler(bot *tgbotapi.BotAPI, userRepo database.UserRepository, subRepo database.SubscriptionRepository, scheduler Scheduler) *Handler {
	return &Handler{
		bot:       bot,
		userRepo:  userRepo,
		subRepo:   subRepo,
		scheduler: scheduler,
	}
}

// HandleUpdate is the main handler for incoming updates.
func (h *Handler) HandleUpdate(update tgbotapi.Update) {
	switch {
	case update.Message != nil:
		h.handleMessage(update.Message)
	case update.CallbackQuery != nil:
		h.handleCallbackQuery(update.CallbackQuery)
	}
}

// handleMessage processes all incoming messages (commands and text).
func (h *Handler) handleMessage(msg *tgbotapi.Message) {
	user, err := h.getOrCreateUser(msg.From)
	if err != nil {
		log.Printf("Error getting or creating user: %v", err)
		return
	}

	if msg.IsCommand() {
		h.handleCommand(msg, user)
		return
	}

	h.handleTextMessage(msg, user)
}

// getOrCreateUser finds a user in the DB or creates a new one.
func (h *Handler) getOrCreateUser(from *tgbotapi.User) (*database.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.userRepo.FindOrCreateUser(ctx, from.ID, from.UserName, from.FirstName, from.LastName)
}

// handleCommand processes bot commands.
func (h *Handler) handleCommand(msg *tgbotapi.Message, user *database.User) {
	ctx := context.Background()
	command := msg.Command()
	topic := strings.TrimSpace(msg.CommandArguments())

	switch command {
	case "start":
		h.handleStart(msg.Chat.ID)
	case "help":
		h.handleHelp(msg.Chat.ID)
	case "subscribe":
		if topic != "" {
			h.handleSubscribe(user, topic, msg.Chat.ID)
		} else {
			h.setUserState(ctx, user.ID, StateAwaitingTopic, msg.Chat.ID)
			h.sendMsg(msg.Chat.ID, "✏️ Введите тему, на которую хотите подписаться.")
		}
	case "unsubscribe":
		if topic != "" {
			h.handleUnsubscribeCommand(ctx, user, topic, msg.Chat.ID)
		} else {
			h.handleUnsubscribeButton(ctx, user, msg.Chat.ID)
		}
	case "subscriptions":
		h.handleSubscriptionsList(ctx, user, msg.Chat.ID)
	case "settings":
		h.handleSettings(msg.Chat.ID)
	default:
		h.sendMsg(msg.Chat.ID, "Неизвестная команда. Используйте /help для списка команд.")
	}
}

// handleTextMessage processes text messages and button clicks.
func (h *Handler) handleTextMessage(msg *tgbotapi.Message, user *database.User) {
	ctx := context.Background()

	// First, check the user's state.
	switch user.State {
	case StateAwaitingTopic:
		h.handleSubscribe(user, msg.Text, msg.Chat.ID)
		h.setUserState(ctx, user.ID, StateDefault, msg.Chat.ID) // Reset state
		return
	case StateAwaitingSearchQuery:
		h.handleSearchNewsQuery(ctx, user, msg.Text, msg.Chat.ID)
		h.setUserState(ctx, user.ID, StateDefault, msg.Chat.ID) // Reset state
		return
	}

	// Then, handle button text.
	switch msg.Text {
	case "➕ Подписаться":
		h.setUserState(ctx, user.ID, StateAwaitingTopic, msg.Chat.ID)
		h.sendMsg(msg.Chat.ID, "✏️ Введите тему, на которую хотите подписаться.")
	case "➖ Отписаться":
		h.handleUnsubscribeButton(ctx, user, msg.Chat.ID)
	case "📋 Мои подписки":
		h.handleSubscriptionsList(ctx, user, msg.Chat.ID)
	case "⚙️ Настройки":
		h.handleSettings(msg.Chat.ID)
	case "📰 Получить новости":
		h.handleGetNewsNow(user)
	case "📃 Новости по темам":
		h.handleNewsByTopics(ctx, user, msg.Chat.ID)
	case "🔄 Сбросить историю":
		h.handleResetHistory(ctx, user, msg.Chat.ID)
	case "🔍 Поиск новостей":
		h.handleSearchNews(ctx, user, msg.Chat.ID)
	case "⭐ Избранное":
		h.handleFavorites(ctx, user, msg.Chat.ID)
	case "❓ Помощь":
		h.handleHelp(msg.Chat.ID)
	default:
		h.sendMsg(msg.Chat.ID, "🤔 Не совсем понял вас. Пожалуйста, используйте кнопки меню или введите команду. Список команд можно посмотреть в /help.")
	}
}

// --- Helper functions for commands and buttons ---

func (h *Handler) handleStart(chatID int64) {
	text := "👋 Привет! Я твой личный бот для отслеживания новостей.\n\n" +
		"Я помогу тебе быть в курсе всех событий по интересующим тебя темам.\n\n" +
		"👇 Просто используй кнопки внизу или команды, чтобы начать."
	h.sendMsg(chatID, text, h.createMainKeyboard())
}

func (h *Handler) handleHelp(chatID int64) {
	helpText := "*Доступные команды и кнопки:*\n\n" +
		"*/start* - ✨ Начало работы с ботом\n" +
		"*/subscribe <тема>* - ➕ Подписаться на новости\n" +
		"*/unsubscribe <тема>* - ➖ Отписаться от новостей\n" +
		"*/subscriptions* - 📋 Показать все ваши активные подписки\n" +
		"*/settings* - ⚙️ Настроить частоту и количество новостей\n" +
		"*/help* - ℹ️ Показать это справочное сообщение\n\n" +
		"*Кнопки в главном меню:*\n" +
		"📰 Получить новости сейчас - мгновенное получение новостей по всем подпискам\n" +
		"📃 Новости по темам - выбор конкретной темы для получения новостей\n" +
		"📋 Мои подписки - управление вашими подписками\n" +
		"🔍 Поиск новостей - поиск новостей по произвольному запросу\n" +
		"⭐ Избранное - просмотр сохраненных вами новостей\n" +
		"🔄 Сбросить историю - очистка истории просмотренных новостей\n" +
		"⚙️ Настройки - изменение частоты и количества новостей\n\n" +
		"*Советы:*\n" +
		"- Для получения новостей по конкретной теме, используйте кнопку 'Новости по темам'\n" +
		"- Для поиска новостей по произвольному запросу, нажмите 'Поиск новостей' и введите интересующий вас запрос"
	h.sendMsg(chatID, helpText)
}

func (h *Handler) handleGetNewsNow(user *database.User) {
	h.sendMsg(user.TelegramID, "🚀 Запускаю поиск свежих новостей по вашим подпискам... Это может занять несколько секунд.")
	go func() {
		processCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		newsSent := h.scheduler.ProcessUser(processCtx, *user, true)
		if newsSent == 0 {
			h.sendMsg(user.TelegramID, "🔍 Свежих новостей по вашим подпискам не найдено.")
		}
	}()
}

func (h *Handler) handleSubscribe(user *database.User, topic string, chatID int64) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		h.sendMsg(chatID, "Вы не ввели тему. Попробуйте снова.")
		return
	}
	topic = strings.ToLower(topic)
	if err := h.subRepo.AddSubscription(context.Background(), user.ID, topic); err != nil {
		h.sendMsg(chatID, fmt.Sprintf("⚠️ Ошибка: не удалось добавить подписку на '%s'. Возможно, вы уже подписаны.", topic))
		log.Printf("Ошибка при добавлении подписки: %v", err)
		return
	}
	h.sendMsg(chatID, fmt.Sprintf("👍 Отлично! Вы подписались на тему: *%s*", topic))
}

func (h *Handler) handleUnsubscribeCommand(ctx context.Context, user *database.User, topic string, chatID int64) {
	topic = strings.ToLower(strings.TrimSpace(topic))
	if err := h.subRepo.RemoveSubscription(ctx, user.ID, topic); err != nil {
		h.sendMsg(chatID, fmt.Sprintf("⚠️ Ошибка: не удалось отписаться от '%s'. Возможно, вы не были подписаны на эту тему.", topic))
		return
	}
	h.sendMsg(chatID, fmt.Sprintf("🗑 Вы успешно отписались от темы: *%s*", topic))
}

func (h *Handler) handleUnsubscribeButton(ctx context.Context, user *database.User, chatID int64) {
	topics, err := h.subRepo.GetUserSubscriptions(ctx, user.ID)
	if err != nil {
		log.Printf("Failed to get user subscriptions: %v", err)
		h.sendMsg(chatID, "Не удалось загрузить ваши подписки. Попробуйте позже.")
		return
	}
	if len(topics) == 0 {
		h.sendMsg(chatID, "У вас нет активных подписок.")
		return
	}
	h.sendMsg(chatID, "Выберите тему, от которой хотите отписаться:", h.createUnsubscribeKeyboard(topics))
}

func (h *Handler) handleSubscriptionsList(ctx context.Context, user *database.User, chatID int64) {
	topics, err := h.subRepo.GetUserSubscriptions(ctx, user.ID)
	if err != nil {
		log.Printf("Ошибка при получении подписок: %v", err)
		h.sendMsg(chatID, "Ошибка при получении списка подписок.")
		return
	}
	if len(topics) == 0 {
		h.sendMsg(chatID, "У вас пока нет подписок. 🤷‍♂️\n\nНажмите '✍️ Подписаться', чтобы добавить свою первую тему!")
	} else {
		var builder strings.Builder
		builder.WriteString("📄 *Ваши текущие подписки:*\n\n")
		for _, topic := range topics {
			builder.WriteString(fmt.Sprintf("• %s\n", topic))
		}
		h.sendMsg(chatID, builder.String())
	}
}

func (h *Handler) handleSettings(chatID int64) {
	text := "Выберите настройки бота:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Частота обновлений", "settings_interval"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Количество новостей", "settings_news_limit"),
		),
	)
	h.sendMsg(chatID, text, keyboard)
}

// --- Callback Handlers ---

func (h *Handler) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	switch {
	case callback.Data == "settings_interval":
		h.handleIntervalSettings(callback)
	case callback.Data == "settings_news_limit":
		h.handleNewsLimitSettings(callback)
	case callback.Data == "settings_back":
		h.handleSettings(callback.Message.Chat.ID)
		h.answerCallback(callback, "")
	case strings.HasPrefix(callback.Data, "interval_"):
		h.handleIntervalCallback(callback)
	case strings.HasPrefix(callback.Data, "news_limit_"):
		h.handleNewsLimitCallback(callback)
	case strings.HasPrefix(callback.Data, "unsubscribe_"):
		h.handleUnsubscribeCallback(callback)
	case strings.HasPrefix(callback.Data, "topic_news_"):
		h.handleTopicNewsCallback(callback)
	case strings.HasPrefix(callback.Data, "add_favorite_") || strings.HasPrefix(callback.Data, "add_fav_"):
		h.handleAddToFavorites(ctx, callback)
	case strings.HasPrefix(callback.Data, "remove_favorite_") || strings.HasPrefix(callback.Data, "rm_fav_"):
		h.handleRemoveFromFavorites(ctx, callback)
	}
}

// Обработчик настроек интервала обновления
func (h *Handler) handleIntervalSettings(callback *tgbotapi.CallbackQuery) {
	text := "Выберите, как часто вы хотите получать новости:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Раз в час", "interval_60"),
			tgbotapi.NewInlineKeyboardButtonData("Раз в 3 часа", "interval_180"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Раз в 6 часов", "interval_360"),
			tgbotapi.NewInlineKeyboardButtonData("Раз в день", "interval_1440"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад", "settings_back"),
		),
	)

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		text,
		keyboard,
	)

	if _, err := h.bot.Send(editMsg); err != nil {
		log.Printf("Ошибка редактирования сообщения: %v", err)
	}
	h.answerCallback(callback, "")
}

// Обработчик настроек количества новостей
func (h *Handler) handleNewsLimitSettings(callback *tgbotapi.CallbackQuery) {
	text := "Выберите, сколько новостей вы хотите получать за один раз:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3 новости", "news_limit_3"),
			tgbotapi.NewInlineKeyboardButtonData("5 новостей", "news_limit_5"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("10 новостей", "news_limit_10"),
			tgbotapi.NewInlineKeyboardButtonData("15 новостей", "news_limit_15"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад", "settings_back"),
		),
	)

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		text,
		keyboard,
	)

	if _, err := h.bot.Send(editMsg); err != nil {
		log.Printf("Ошибка редактирования сообщения: %v", err)
	}
	h.answerCallback(callback, "")
}

// Обработчик выбора интервала
func (h *Handler) handleIntervalCallback(callback *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	user, err := h.getOrCreateUser(callback.From)
	if err != nil {
		log.Printf("Ошибка поиска пользователя %d: %v", callback.From.ID, err)
		h.answerCallback(callback, "Произошла ошибка.")
		return
	}

	intervalStr := strings.TrimPrefix(callback.Data, "interval_")
	interval, _ := strconv.Atoi(intervalStr)
	if err := h.userRepo.UpdateUserNotificationInterval(ctx, user.ID, uint(interval)); err != nil {
		log.Printf("Ошибка обновления настроек для пользователя %d: %v", user.ID, err)
		h.answerCallback(callback, "Не удалось обновить настройки.")
		return
	}

	responseText := fmt.Sprintf("Интервал обновления установлен на %d минут.", interval)
	h.answerCallback(callback, responseText)

	// Возвращаемся в меню настроек
	h.handleSettings(callback.Message.Chat.ID)
}

// Обработчик выбора количества новостей
func (h *Handler) handleNewsLimitCallback(callback *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	user, err := h.getOrCreateUser(callback.From)
	if err != nil {
		log.Printf("Ошибка поиска пользователя %d: %v", callback.From.ID, err)
		h.answerCallback(callback, "Произошла ошибка.")
		return
	}

	limitStr := strings.TrimPrefix(callback.Data, "news_limit_")
	limit, _ := strconv.Atoi(limitStr)
	if err := h.userRepo.UpdateUserNewsLimit(ctx, user.ID, uint(limit)); err != nil {
		log.Printf("Ошибка обновления настроек для пользователя %d: %v", user.ID, err)
		h.answerCallback(callback, "Не удалось обновить настройки.")
		return
	}

	responseText := fmt.Sprintf("Количество новостей установлено на %d.", limit)
	h.answerCallback(callback, responseText)

	// Возвращаемся в меню настроек
	h.handleSettings(callback.Message.Chat.ID)
}

// Обработчик кнопки "Новости по темам"
func (h *Handler) handleNewsByTopics(ctx context.Context, user *database.User, chatID int64) {
	// Получаем список подписок пользователя
	topics, err := h.subRepo.GetUserSubscriptions(ctx, user.ID)
	if err != nil {
		log.Printf("Ошибка получения подписок пользователя: %v", err)
		h.sendMsg(chatID, "Произошла ошибка при получении ваших подписок.")
		return
	}

	if len(topics) == 0 {
		h.sendMsg(chatID, "У вас пока нет подписок. Используйте команду /subscribe или кнопку 'Подписаться', чтобы добавить темы.")
		return
	}

	// Создаем инлайн кнопки для каждой темы
	text := "Выберите тему, по которой хотите получить новости:"

	// Создаем строки кнопок, по 2 кнопки в строке
	var rows [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	for i, topic := range topics {
		// Создаем кнопку с темой
		button := tgbotapi.NewInlineKeyboardButtonData(topic, "topic_news_"+topic)
		currentRow = append(currentRow, button)

		// Если у нас 2 кнопки в строке или это последняя тема, добавляем строку в клавиатуру
		if len(currentRow) == 2 || i == len(topics)-1 {
			rows = append(rows, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.sendMsg(chatID, text, keyboard)
}

// Обработчик нажатия на кнопку с темой для получения новостей
func (h *Handler) handleTopicNewsCallback(callback *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	user, err := h.getOrCreateUser(callback.From)
	if err != nil {
		log.Printf("Ошибка поиска пользователя: %v", err)
		h.answerCallback(callback, "Произошла ошибка.")
		return
	}

	// Получаем тему из данных кнопки
	topic := strings.TrimPrefix(callback.Data, "topic_news_")

	// Отвечаем на колбэк, чтобы убрать индикатор загрузки
	h.answerCallback(callback, "Ищу новости по теме '"+topic+"'...")

	// Отправляем сообщение о начале поиска
	h.sendMsg(callback.Message.Chat.ID, "🔍 Ищу новости по теме '"+topic+"'... Это может занять несколько секунд.")

	// Запускаем поиск новостей в отдельной горутине
	go func() {
		// Получаем новости по теме
		articles, err := h.fetchNewsForTopic(ctx, user, topic)
		if err != nil {
			log.Printf("Ошибка получения новостей по теме '%s': %v", topic, err)

			// Проверяем на ошибку лимита запросов
			if strings.Contains(err.Error(), "request limit") || strings.Contains(err.Error(), "rate limit") {
				h.sendMsg(callback.Message.Chat.ID, "❗ Достигнут лимит запросов к API. Пожалуйста, попробуйте позже.")
			} else {
				h.sendMsg(callback.Message.Chat.ID, "Произошла ошибка при получении новостей. Попробуйте другую тему или повторите запрос позже.")
			}
			return
		}

		if len(articles) == 0 {
			h.sendMsg(callback.Message.Chat.ID, "🔍 Свежих новостей по теме '"+topic+"' не найдено.")
			return
		}

		// Отправляем новости
		h.sendMsg(callback.Message.Chat.ID, "📰 Новости по теме '"+topic+"':")

		// Ограничиваем количество новостей по настройкам пользователя
		newsLimit := int(user.NewsLimit)
		if newsLimit <= 0 {
			newsLimit = 5 // Значение по умолчанию
		}

		articlesToSend := articles
		if len(articles) > newsLimit {
			articlesToSend = articles[:newsLimit]
		}

		for _, article := range articlesToSend {
			// Используем метод отправки статьи с кнопкой "В избранное"
			if err := h.sendArticleWithFavoriteButton(ctx, callback.Message.Chat.ID, user.ID, article); err != nil {
				log.Printf("Ошибка отправки новости: %v", err)
				continue
			}
		}
	}()
}

// fetchNewsForTopic получает новости по теме, фильтрует их по дате и уже отправленным
func (h *Handler) fetchNewsForTopic(ctx context.Context, user *database.User, topic string) ([]fetcher.Article, error) {
	// Получаем новости по теме
	articles, err := h.scheduler.FetchNewsForTopic(ctx, topic)
	if err != nil {
		return nil, err
	}

	// Фильтруем статьи, которые уже были отправлены пользователю
	return h.filterSentArticles(ctx, user.ID, articles)
}

// filterSentArticles фильтрует статьи, которые уже были отправлены пользователю
func (h *Handler) filterSentArticles(ctx context.Context, userID uint, articles []fetcher.Article) ([]fetcher.Article, error) {
	// Фильтруем статьи, которые уже были отправлены пользователю
	freshArticles := []fetcher.Article{}
	for _, article := range articles {
		// Проверяем, была ли статья уже отправлена
		isSent, err := h.scheduler.IsArticleSent(ctx, userID, article.URL)
		if err != nil {
			log.Printf("Ошибка проверки отправленной статьи: %v", err)
			continue
		}

		// Если статья еще не была отправлена, добавляем ее в список
		if !isSent {
			freshArticles = append(freshArticles, article)
		}
	}

	return freshArticles, nil
}

// formatArticleMessage создает красиво отформатированное HTML-сообщение для новостной статьи
func (h *Handler) formatArticleMessage(article fetcher.Article) string {
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

	// Очищаем текст от некорректных символов
	title := h.sanitizeText(article.Title)
	description = h.sanitizeText(description)
	sourceName = h.sanitizeText(sourceName)

	// Создаем HTML-сообщение с форматированием
	message := fmt.Sprintf(
		"<b>%s</b>\n\n"+ // Заголовок жирным шрифтом
			"%s\n\n"+ // Описание
			"<i>📰 Источник: %s</i>\n"+ // Источник курсивом
			"<i>📅 Опубликовано: %s</i>\n\n"+ // Дата публикации курсивом
			"<a href=\"%s\">Читать полностью →</a>", // Ссылка на статью
		title,
		description,
		sourceName,
		publishedDate,
		article.URL,
	)

	return message
}

// handleSearchNews обрабатывает нажатие на кнопку "Поиск новостей"
func (h *Handler) handleSearchNews(ctx context.Context, user *database.User, chatID int64) {
	// Устанавливаем состояние ожидания поискового запроса
	h.setUserState(ctx, user.ID, StateAwaitingSearchQuery, chatID)

	// Отправляем сообщение с инструкцией
	h.sendMsg(chatID, "🔍 Введите поисковый запрос для поиска новостей.\n\nНапример: 'искусственный интеллект', 'новые технологии', 'космос' и т.д.")
}

// handleSearchNewsQuery обрабатывает поисковый запрос пользователя
func (h *Handler) handleSearchNewsQuery(ctx context.Context, user *database.User, query string, chatID int64) {
	// Проверяем, что запрос не пустой
	if strings.TrimSpace(query) == "" {
		h.sendMsg(chatID, "❌ Поисковый запрос не может быть пустым. Пожалуйста, введите запрос для поиска новостей.")
		return
	}

	// Отправляем сообщение о начале поиска
	h.sendMsg(chatID, fmt.Sprintf("🔍 Ищу новости по запросу '%s'... Это может занять несколько секунд.", query))

	// Запускаем поиск в отдельной горутине
	go func() {
		// Получаем новости по запросу
		articles, err := h.scheduler.SearchNews(ctx, query)
		if err != nil {
			log.Printf("Ошибка поиска новостей по запросу '%s': %v", query, err)
			if strings.Contains(err.Error(), "request limit") || strings.Contains(err.Error(), "rate limit") {
				h.sendMsg(chatID, "❗ Достигнут лимит запросов к API. Пожалуйста, попробуйте позже.")
			} else {
				h.sendMsg(chatID, "Произошла ошибка при поиске новостей. Попробуйте другой запрос или повторите позже.")
			}
			return
		}

		// Проверяем, что найдены новости
		if len(articles) == 0 {
			h.sendMsg(chatID, fmt.Sprintf("🔍 Новостей по запросу '%s' не найдено. Попробуйте изменить запрос.", query))
			return
		}

		// Фильтруем новости, которые уже были отправлены пользователю
		freshArticles, err := h.filterSentArticles(ctx, user.ID, articles)
		if err != nil {
			log.Printf("Ошибка фильтрации отправленных статей: %v", err)
			h.sendMsg(chatID, "Произошла ошибка при обработке результатов. Пожалуйста, попробуйте позже.")
			return
		}

		// Если после фильтрации не осталось новостей, сообщаем пользователю
		if len(freshArticles) == 0 {
			h.sendMsg(chatID, fmt.Sprintf("🔍 По запросу '%s' найдены только новости, которые вы уже получали ранее. Попробуйте другой запрос или сбросьте историю.", query))
			return
		}

		// Отправляем заголовок с результатами
		h.sendMsg(chatID, fmt.Sprintf("📰 Результаты поиска по запросу '%s':", query))

		// Ограничиваем количество отправляемых новостей
		newsLimit := int(user.NewsLimit)
		if newsLimit <= 0 {
			newsLimit = 5 // Значение по умолчанию
		}

		// Если новостей больше, чем лимит, берем только первые newsLimit
		articlesToSend := freshArticles
		if len(freshArticles) > newsLimit {
			articlesToSend = freshArticles[:newsLimit]
		}

		// Отправляем новости
		for _, article := range articlesToSend {
			// Используем метод отправки статьи с кнопкой "В избранное"
			if err := h.sendArticleWithFavoriteButton(ctx, chatID, user.ID, article); err != nil {
				log.Printf("Ошибка отправки новости: %v", err)
				continue
			}

			// Помечаем статью как отправленную
			if err := h.scheduler.MarkArticleAsSent(ctx, user.ID, article.URL); err != nil {
				log.Printf("Ошибка при маркировке статьи как отправленной: %v", err)
			}
		}

		// Если есть еще новости, которые не были отправлены из-за лимита, сообщаем пользователю
		if len(freshArticles) > newsLimit {
			h.sendMsg(chatID, fmt.Sprintf("ℹ️ Показано %d из %d найденных новостей. Чтобы увидеть больше новостей, измените лимит в настройках или используйте более конкретный запрос.", newsLimit, len(freshArticles)))
		}
	}()
}

// handleResetHistory обрабатывает нажатие на кнопку "Сбросить историю"
func (h *Handler) handleResetHistory(ctx context.Context, user *database.User, chatID int64) {
	// Отправляем сообщение о начале процесса
	h.sendMsg(chatID, "🔄 Сбрасываю историю отправленных новостей... Это может занять несколько секунд.")

	// Запускаем сброс истории в отдельной горутине
	go func() {
		// Сбрасываем историю отправленных статей
		err := h.scheduler.ResetSentArticlesHistory(ctx, user.ID)
		if err != nil {
			log.Printf("Ошибка сброса истории отправленных статей: %v", err)
			h.sendMsg(chatID, "❌ Произошла ошибка при сбросе истории. Пожалуйста, попробуйте позже.")
			return
		}

		// Отправляем сообщение об успешном сбросе
		h.sendMsg(chatID, "✅ История отправленных новостей успешно сброшена! Теперь вы будете получать новости, которые уже были отправлены ранее.")
	}()
}

func (h *Handler) handleUnsubscribeCallback(callback *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	user, err := h.getOrCreateUser(callback.From)
	if err != nil {
		log.Printf("Ошибка поиска пользователя при отписке: %v", err)
		h.answerCallback(callback, "Произошла ошибка.")
		return
	}

	topicToUnsubscribe := strings.TrimPrefix(callback.Data, "unsubscribe_")
	if err := h.subRepo.RemoveSubscription(ctx, user.ID, topicToUnsubscribe); err != nil {
		h.answerCallback(callback, "Не удалось отписаться.")
		return
	}

	responseText := fmt.Sprintf("Вы отписались от темы: %s", topicToUnsubscribe)
	h.answerCallback(callback, responseText)
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, responseText)
	newKeyboard := h.removeButtonFromKeyboard(callback.Message.ReplyMarkup, callback.Data)
	editMsg.ReplyMarkup = newKeyboard
	if _, err := h.bot.Send(editMsg); err != nil {
		log.Printf("Ошибка редактирования сообщения: %v", err)
	}
}

// --- Helper functions ---

func (h *Handler) createMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		// Первый ряд: Основные функции получения новостей
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📰 Получить новости"),
			tgbotapi.NewKeyboardButton("📃 Новости по темам"),
			tgbotapi.NewKeyboardButton("🔍 Поиск новостей"),
		),
		// Второй ряд: Управление подписками
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Подписаться"),
			tgbotapi.NewKeyboardButton("➖ Отписаться"),
			tgbotapi.NewKeyboardButton("📋 Мои подписки"),
		),
		// Третий ряд: Избранное и дополнительные функции
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⭐ Избранное"),
			tgbotapi.NewKeyboardButton("🔄 Сбросить историю"),
		),
		// Четвертый ряд: Настройки и помощь
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
			tgbotapi.NewKeyboardButton("❓ Помощь"),
		),
	)
}

func (h *Handler) sendMsg(chatID int64, text string, markup ...interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if len(markup) > 0 {
		msg.ReplyMarkup = markup[0]
	}
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Ошибка при отправке сообщения: %v", err)
	}
}

func (h *Handler) setUserState(ctx context.Context, userID uint, state string, chatID int64) {
	if err := h.userRepo.SetUserState(ctx, userID, state); err != nil {
		log.Printf("Failed to set user state for user %d: %v", userID, err)
		h.sendMsg(chatID, "Произошла внутренняя ошибка. Попробуйте еще раз.")
	}
}

func (h *Handler) createUnsubscribeKeyboard(topics []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, topic := range topics {
		button := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("❌ %s", topic), "unsubscribe_"+topic)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (h *Handler) answerCallback(callback *tgbotapi.CallbackQuery, text string) {
	answer := tgbotapi.NewCallback(callback.ID, text)
	if _, err := h.bot.Request(answer); err != nil {
		log.Printf("Ошибка ответа на callback: %v", err)
	}
}

func (h *Handler) removeButtonFromKeyboard(keyboard *tgbotapi.InlineKeyboardMarkup, buttonData string) *tgbotapi.InlineKeyboardMarkup {
	if keyboard == nil {
		return nil
	}
	var newRows [][]tgbotapi.InlineKeyboardButton
	for _, row := range keyboard.InlineKeyboard {
		var newRow []tgbotapi.InlineKeyboardButton
		for _, button := range row {
			if button.CallbackData == nil || *button.CallbackData != buttonData {
				newRow = append(newRow, button)
			}
		}
		if len(newRow) > 0 {
			newRows = append(newRows, newRow)
		}
	}
	if len(newRows) == 0 {
		return nil // If no buttons are left, remove the keyboard
	}
	newMarkup := tgbotapi.NewInlineKeyboardMarkup(newRows...)
	return &newMarkup
}

// sanitizeText очищает текст от некорректных символов для безопасной отправки в Telegram API
func (h *Handler) sanitizeText(text string) string {
	// Используем утилитарную функцию из пакета utils
	return utils.SanitizeText(text)
}
