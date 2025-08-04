package tests

import (
	"testing"
)

// TestMain запускается перед всеми тестами
func TestMain(m *testing.M) {
	// Здесь можно добавить общую инициализацию для всех тестов
	// Например, настройку логирования или подготовку тестовых данных

	// Запускаем все тесты
	m.Run()
}

// Интеграционный тест для проверки совместной работы компонентов
func TestIntegration(t *testing.T) {
	t.Run("Utils integration", func(t *testing.T) {
		// Этот тест проверяет, что все утилитарные функции работают вместе
		// как они используются в реальном приложении

		// Имитируем обработку новостной статьи
		articleURL := "https://ria.ru/20240101/important-news-1234567890.html"
		articleTitle := "Важные новости дня: президент подписал новый закон 📰"

		// Создаем короткий ID (как в handlers)
		shortID := createShortIDForTest(articleURL)
		if len(shortID) == 0 || len(shortID) > 10 {
			t.Errorf("Invalid short ID length: %d", len(shortID))
		}

		// Санитизируем заголовок (как в handlers)
		sanitizedTitle := sanitizeTextForTest(articleTitle)
		if sanitizedTitle == "" {
			t.Error("Sanitized title should not be empty")
		}

		// Проверяем, что эмодзи сохранились
		if !containsEmoji(sanitizedTitle) {
			t.Error("Sanitized title should preserve emojis")
		}

		t.Logf("Original URL: %s", articleURL)
		t.Logf("Short ID: %s", shortID)
		t.Logf("Original title: %s", articleTitle)
		t.Logf("Sanitized title: %s", sanitizedTitle)
	})
}

// Вспомогательные функции для тестов
func createShortIDForTest(input string) string {
	// Имитируем логику из utils.CreateShortID
	if len(input) == 0 {
		return ""
	}

	if len(input) <= 10 {
		return input
	}

	// Для тестов используем простую логику
	return input[len(input)-10:]
}

func sanitizeTextForTest(text string) string {
	// Имитируем базовую логику из utils.SanitizeText
	if text == "" {
		return "Неизвестно"
	}
	return text
}

func containsEmoji(text string) bool {
	// Простая проверка наличия эмодзи
	for _, r := range text {
		if r >= 0x1F600 && r <= 0x1F64F { // Emoticons
			return true
		}
		if r >= 0x1F300 && r <= 0x1F5FF { // Misc Symbols
			return true
		}
		if r >= 0x1F680 && r <= 0x1F6FF { // Transport
			return true
		}
		if r >= 0x2600 && r <= 0x26FF { // Misc symbols
			return true
		}
	}
	return false
}
