package handlers

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/fetcher"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/utils"
)

// sendArticleWithFavoriteButton отправляет новостную статью с кнопкой "В избранное"
func (h *Handler) sendArticleWithFavoriteButton(ctx context.Context, chatID int64, userID uint, article fetcher.Article) error {
	// Форматируем сообщение
	messageText := h.formatArticleMessage(article)

	// Проверяем, находится ли статья в избранном
	isFavorite, err := h.scheduler.IsFavoriteArticle(ctx, userID, article.URL)
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
	sanitizedText := h.sanitizeText(messageText)

	// Отправляем сообщение с клавиатурой
	msg := tgbotapi.NewMessage(chatID, sanitizedText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = false
	msg.ReplyMarkup = keyboard

	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки новости: %v", err)
		return err
	}

	return nil
}
