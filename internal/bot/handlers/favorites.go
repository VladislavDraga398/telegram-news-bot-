package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/database"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/fetcher"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/utils"
)

// handleFavorites обрабатывает нажатие на кнопку "Избранное".
func (h *Handler) handleFavorites(ctx context.Context, user *database.User, chatID int64) {
	h.sendMsg(chatID, "🔍 Получаю список избранных новостей...")

	// Получаем избранные новости пользователя
	favorites, err := h.scheduler.GetUserFavoriteArticles(ctx, user.ID)
	if err != nil {
		log.Printf("Ошибка получения избранных новостей: %v", err)
		h.sendMsg(chatID, "❌ Произошла ошибка при получении избранных новостей. Пожалуйста, попробуйте позже.")
		return
	}

	if len(favorites) == 0 {
		h.sendMsg(chatID, "📭 У вас пока нет избранных новостей. Чтобы добавить новость в избранное, нажмите на кнопку '⭐ В избранное' под новостью.")
		return
	}

	// Отправляем список избранных новостей
	h.sendMsg(chatID, fmt.Sprintf("📚 Ваши избранные новости (%d):", len(favorites)))

	// Отправляем каждую избранную новость
	for _, favorite := range favorites {
		// Форматируем дату публикации
		publishedDate := favorite.PublishedAt.Format("02.01.2006 15:04")

		// Очищаем текст от некорректных символов
		title := h.sanitizeText(favorite.Title)
		source := h.sanitizeText(favorite.Source)

		// Создаем сообщение с информацией о новости
		messageText := fmt.Sprintf(
			"<b>%s</b>\n\n"+
				"<i>Источник: %s</i>\n"+
				"<i>Опубликовано: %s</i>\n\n"+
				"<a href=\"%s\">Читать полностью</a>",
			title,
			source,
			publishedDate,
			favorite.ArticleURL,
		)

		// Создаем уникальный идентификатор для кнопки на основе хеша URL
		// Используем только последние 10 символов URL для создания короткого идентификатора
		urlLen := len(favorite.ArticleURL)
		shortID := favorite.ArticleURL
		if urlLen > 10 {
			shortID = favorite.ArticleURL[urlLen-10:]
		}

		// Создаем клавиатуру с кнопкой для удаления из избранного
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Удалить из избранного", "rm_fav_"+shortID),
			),
		)

		// Отправляем сообщение с клавиатурой
		msg := tgbotapi.NewMessage(chatID, messageText)
		msg.ParseMode = tgbotapi.ModeHTML
		msg.DisableWebPagePreview = false
		msg.ReplyMarkup = keyboard

		if _, err := h.bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки избранной новости: %v", err)
		}
	}
}

// handleAddToFavorites обрабатывает добавление новости в избранное.
func (h *Handler) handleAddToFavorites(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	// Получаем URL статьи из данных callback
	var articleURL string
	if strings.HasPrefix(callback.Data, "add_fav_") {
		// Получаем короткий идентификатор
		shortID := callback.Data[len("add_fav_"):]

		// Ищем полный URL в сообщении
		// Ищем в entities ссылку на статью
		for _, entity := range callback.Message.Entities {
			if entity.Type == "text_link" {
				// Берем первую ссылку, так как обычно это и есть ссылка на статью
				articleURL = entity.URL
				break
			}
		}

		if articleURL == "" {
			log.Printf("Не удалось найти полный URL для короткого идентификатора: %s", shortID)
			h.answerCallback(callback, "Произошла ошибка при добавлении в избранное.")
			return
		}
	} else {
		// Старый формат с полным URL
		articleURL = callback.Data[len("add_favorite_"):]
	}

	// Получаем пользователя
	user, err := h.getOrCreateUser(callback.From)
	if err != nil {
		log.Printf("Ошибка поиска пользователя: %v", err)
		h.answerCallback(callback, "Произошла ошибка.")
		return
	}

	// Проверяем, добавлена ли уже статья в избранное
	isFavorite, err := h.scheduler.IsFavoriteArticle(ctx, user.ID, articleURL)
	if err != nil {
		log.Printf("Ошибка проверки избранной статьи: %v", err)
		h.answerCallback(callback, "Произошла ошибка.")
		return
	}

	if isFavorite {
		h.answerCallback(callback, "Эта статья уже в избранном.")
		return
	}

	// Получаем информацию о статье из сообщения
	messageText := callback.Message.Text
	messageEntities := callback.Message.Entities

	// Извлекаем заголовок статьи (первая строка сообщения)
	title := messageText
	if len(messageText) > 50 {
		title = messageText[:50] + "..."
	}

	// Извлекаем источник статьи (если есть)
	source := "Неизвестный источник"
	for _, entity := range messageEntities {
		if entity.Type == "text_link" && entity.URL == articleURL {
			source = messageText[entity.Offset : entity.Offset+entity.Length]
			break
		}
	}

	// Добавляем статью в избранное
	article := fetcher.Article{
		URL:         articleURL,
		Title:       title,
		Source:      fetcher.Source{Name: source},
		PublishedAt: time.Now(),
	}

	if err := h.scheduler.AddFavoriteArticle(ctx, user.ID, article); err != nil {
		log.Printf("Ошибка добавления статьи в избранное: %v", err)
		h.answerCallback(callback, "Произошла ошибка при добавлении в избранное.")
		return
	}

	// Создаем короткий идентификатор для URL статьи
	shortID := utils.CreateShortID(articleURL)

	// Обновляем клавиатуру сообщения, заменяя кнопку "В избранное" на "Удалить из избранного"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Удалить из избранного", "rm_fav_"+shortID),
		),
	)

	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		keyboard,
	)

	if _, err := h.bot.Send(editMsg); err != nil {
		log.Printf("Ошибка обновления клавиатуры: %v", err)
	}

	h.answerCallback(callback, "✅ Статья добавлена в избранное!")
}

// handleRemoveFromFavorites обрабатывает удаление новости из избранного.
func (h *Handler) handleRemoveFromFavorites(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	// Получаем идентификатор статьи из данных callback
	var articleID string

	// Проверяем формат данных callback
	if strings.HasPrefix(callback.Data, "rm_fav_") {
		// Новый формат с коротким идентификатором
		articleID = callback.Data[len("rm_fav_"):]
	} else if strings.HasPrefix(callback.Data, "remove_favorite_") {
		// Старый формат с полным URL
		articleID = callback.Data[len("remove_favorite_"):]
	} else {
		// Неизвестный формат
		log.Printf("Неизвестный формат данных callback: %s", callback.Data)
		h.answerCallback(callback, "Произошла ошибка.")
		return
	}

	// Получаем пользователя
	user, err := h.getOrCreateUser(callback.From)
	if err != nil {
		log.Printf("Ошибка поиска пользователя: %v", err)
		h.answerCallback(callback, "Произошла ошибка.")
		return
	}

	// Если мы используем короткий идентификатор, нам нужно найти полный URL статьи
	if strings.HasPrefix(callback.Data, "rm_fav_") {
		// Получаем список всех избранных статей пользователя
		favorites, err := h.scheduler.GetUserFavoriteArticles(ctx, user.ID)
		if err != nil {
			log.Printf("Ошибка получения избранных статей: %v", err)
			h.answerCallback(callback, "Произошла ошибка при удалении из избранного.")
			return
		}

		// Ищем статью по короткому идентификатору
		found := false
		for _, favorite := range favorites {
			urlLen := len(favorite.ArticleURL)
			shortID := favorite.ArticleURL
			if urlLen > 10 {
				shortID = favorite.ArticleURL[urlLen-10:]
			}

			if shortID == articleID {
				// Нашли статью, удаляем ее по полному URL
				articleID = favorite.ArticleURL
				found = true
				break
			}
		}

		if !found {
			log.Printf("Не удалось найти статью по короткому идентификатору: %s", articleID)
			h.answerCallback(callback, "Произошла ошибка при удалении из избранного.")
			return
		}
	}

	// Удаляем статью из избранного
	if err := h.scheduler.RemoveFavoriteArticle(ctx, user.ID, articleID); err != nil {
		log.Printf("Ошибка удаления статьи из избранного: %v", err)
		h.answerCallback(callback, "Произошла ошибка при удалении из избранного.")
		return
	}

	// Если удаление происходит из списка избранных новостей, удаляем сообщение
	if callback.Message.ReplyMarkup != nil && len(callback.Message.ReplyMarkup.InlineKeyboard) > 0 {
		data := callback.Message.ReplyMarkup.InlineKeyboard[0][0].CallbackData
		if data != nil && len(*data) > len("remove_favorite_") && (*data)[:len("remove_favorite_")] == "remove_favorite_" {
			deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
			if _, err := h.bot.Send(deleteMsg); err != nil {
				log.Printf("Ошибка удаления сообщения: %v", err)
			}
			h.answerCallback(callback, "✅ Статья удалена из избранного!")
			return
		}
	}

	// Обновляем клавиатуру сообщения, заменяя кнопку "Удалить из избранного" на "В избранное"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ В избранное", "add_favorite_"+articleID),
		),
	)

	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		keyboard,
	)

	if _, err := h.bot.Send(editMsg); err != nil {
		log.Printf("Ошибка обновления клавиатуры: %v", err)
	}

	h.answerCallback(callback, "✅ Статья удалена из избранного!")
}
