package handlers_test

import (
	"testing"

	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/utils"
)

// Тестируем интеграцию handlers с utils
func TestHandlerTextSanitization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal news title",
			input:    "Важные новости дня",
			expected: "Важные новости дня",
		},
		{
			name:     "Title with emojis",
			input:    "🔥 Горячие новости 📰",
			expected: "🔥 Горячие новости 📰",
		},
		{
			name:     "Empty title",
			input:    "",
			expected: "Неизвестно",
		},
		{
			name:     "Title with invalid UTF-8",
			input:    "Новости\x00с\x01проблемами",
			expected: "Новости с проблемами",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SanitizeText(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeText() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestShortIDGeneration(t *testing.T) {
	// Тестируем создание коротких ID для различных URL новостей
	newsURLs := []string{
		"https://ria.ru/20240101/news-1234567890.html",
		"https://tass.ru/obschestvo/12345678",
		"https://lenta.ru/news/2024/01/01/important/",
		"https://interfax.ru/russia/123456",
	}

	ids := make(map[string]bool)

	for _, url := range newsURLs {
		shortID := utils.CreateShortID(url)

		// Проверяем длину
		if len(shortID) > 10 {
			t.Errorf("Short ID too long: %s (length: %d)", shortID, len(shortID))
		}

		// Проверяем уникальность
		if ids[shortID] {
			t.Errorf("Duplicate short ID generated: %s", shortID)
		}
		ids[shortID] = true

		// Проверяем, что ID не пустой
		if shortID == "" {
			t.Errorf("Empty short ID generated for URL: %s", url)
		}
	}
}

func TestCallbackDataFormat(t *testing.T) {
	// Тестируем формат callback данных для кнопок
	testURL := "https://example.com/very/long/news/article/url"
	shortID := utils.CreateShortID(testURL)

	// Тестируем различные префиксы callback данных
	prefixes := []string{"add_fav_", "rm_fav_"}

	for _, prefix := range prefixes {
		callbackData := prefix + shortID

		// Проверяем, что callback данные не превышают лимит Telegram (64 байта)
		if len(callbackData) > 64 {
			t.Errorf("Callback data too long: %s (length: %d)", callbackData, len(callbackData))
		}

		// Проверяем, что данные содержат правильный префикс
		if callbackData[:len(prefix)] != prefix {
			t.Errorf("Callback data should start with %s, got: %s", prefix, callbackData)
		}
	}
}

// Бенчмарк для проверки производительности обработки текста
func BenchmarkTextProcessing(b *testing.B) {
	longText := `Это очень длинный текст новости с различными символами: 
	кириллица, латиница, цифры 123, эмодзи 😀, знаки препинания... 
	и другие символы! Этот текст имитирует реальную новостную статью 
	с заголовком и содержанием.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.SanitizeText(longText)
	}
}

func BenchmarkShortIDGeneration(b *testing.B) {
	longURL := "https://example.com/very/long/news/article/url/with/many/segments/and/parameters?param1=value1&param2=value2&param3=value3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.CreateShortID(longURL)
	}
}
