package utils_test

import (
	"strings"
	"testing"

	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/utils"
)

func TestCreateShortID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Short string",
			input:    "short",
			expected: "short",
		},
		{
			name:     "Exactly 10 characters",
			input:    "1234567890",
			expected: "1234567890",
		},
		{
			name:  "Long URL",
			input: "https://example.com/very/long/path/to/article",
			// Должен возвращать последние 10 символов MD5-хеша
		},
		{
			name:  "Russian URL",
			input: "https://example.com/статья",
			// Должен корректно обрабатывать кириллицу
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.CreateShortID(tt.input)

			// Проверяем, что результат не превышает 10 символов
			if len(result) > 10 {
				t.Errorf("CreateShortID() returned string longer than 10 characters: %d", len(result))
			}

			// Для пустой строки ожидаем пустой результат
			if tt.input == "" && result != "" {
				t.Errorf("CreateShortID() for empty string should return empty string, got: %s", result)
			}

			// Для коротких строк ожидаем исходную строку
			if len(tt.input) <= 10 && tt.input != "" && result != tt.input {
				t.Errorf("CreateShortID() for short string should return original string, got: %s, expected: %s", result, tt.input)
			}

			// Для длинных строк ожидаем хеш
			if len(tt.input) > 10 && len(result) != 10 {
				t.Errorf("CreateShortID() for long string should return 10-character hash, got length: %d", len(result))
			}
		})
	}
}

func TestCreateShortIDConsistency(t *testing.T) {
	// Тест на консистентность - одинаковый вход должен давать одинаковый результат
	input := "https://example.com/very/long/path/to/article/with/many/segments"

	result1 := utils.CreateShortID(input)
	result2 := utils.CreateShortID(input)

	if result1 != result2 {
		t.Errorf("CreateShortID() should be consistent, got %s and %s for same input", result1, result2)
	}
}

func TestCreateShortIDUniqueness(t *testing.T) {
	// Тест на уникальность - разные входы должны давать разные результаты
	inputs := []string{
		"https://example.com/article1",
		"https://example.com/article2",
		"https://example.com/different/path",
		"https://another-domain.com/article",
	}

	results := make(map[string]bool)

	for _, input := range inputs {
		result := utils.CreateShortID(input)
		if results[result] {
			t.Errorf("CreateShortID() produced duplicate result %s for different inputs", result)
		}
		results[result] = true
	}
}

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "Неизвестно",
		},
		{
			name:     "Normal text",
			input:    "Обычный текст",
			expected: "Обычный текст",
		},
		{
			name:     "Text with emojis",
			input:    "Текст с эмодзи 😀 и символами",
			expected: "Текст с эмодзи 😀 и символами",
		},
		{
			name:     "Text with invalid characters",
			input:    "Текст\x00с\x01недопустимыми\x02символами",
			expected: "Текст с недопустимыми символами",
		},
		{
			name:     "Multiple spaces",
			input:    "Текст    с     множественными    пробелами",
			expected: "Текст с множественными пробелами",
		},
		{
			name:     "Only spaces",
			input:    "   ",
			expected: "Неизвестно",
		},
		{
			name:     "Mixed languages",
			input:    "Mixed русский English текст",
			expected: "Mixed русский English текст",
		},
		{
			name:     "Special punctuation",
			input:    "Текст с «кавычками» и тире — точками…",
			expected: "Текст с «кавычками» и тире — точками…",
		},
		{
			name:     "Numbers and symbols",
			input:    "Цена: 1000₽, скидка 50%",
			expected: "Цена: 1000₽, скидка 50%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SanitizeText(tt.input)

			// Проверяем, что результат не пустой (кроме случая, когда ожидается "Неизвестно")
			if result == "" {
				t.Errorf("SanitizeText() should never return empty string, got empty for input: %q", tt.input)
			}

			// Проверяем, что в результате нет множественных пробелов
			if strings.Contains(result, "  ") {
				t.Errorf("SanitizeText() should not contain multiple spaces, got: %q", result)
			}

			// Для некоторых тестов проверяем точное соответствие
			if tt.expected != "" && result != tt.expected {
				t.Errorf("SanitizeText() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeTextUTF8Validity(t *testing.T) {
	// Тест для проверки, что результат всегда является валидным UTF-8
	invalidUTF8 := []byte{0xff, 0xfe, 0xfd}
	input := "Текст с невалидным UTF-8: " + string(invalidUTF8)

	result := utils.SanitizeText(input)

	// Проверяем, что результат является валидным UTF-8
	if !isValidUTF8(result) {
		t.Errorf("SanitizeText() should always return valid UTF-8, got invalid UTF-8: %q", result)
	}
}

func TestSanitizeTextPreservesImportantContent(t *testing.T) {
	// Тест для проверки, что важный контент сохраняется
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "News headline",
			input: "Важные новости: президент подписал закон о цифровых правах",
		},
		{
			name:  "Article with quotes",
			input: "Эксперт заявил: «Это важное решение для экономики»",
		},
		{
			name:  "Text with percentages",
			input: "Рост составил 15% по сравнению с прошлым годом",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SanitizeText(tt.input)

			// Проверяем, что основные слова сохранились
			words := strings.Fields(tt.input)
			for _, word := range words {
				// Пропускаем очень короткие слова и знаки препинания
				if len(word) > 2 && !strings.ContainsAny(word, ".,!?:;") {
					if !strings.Contains(result, word) {
						t.Errorf("SanitizeText() should preserve important word %q, result: %q", word, result)
					}
				}
			}
		})
	}
}

// isValidUTF8 проверяет, является ли строка валидным UTF-8
func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' {
			// Найден символ замещения, что может указывать на невалидный UTF-8
			return false
		}
	}
	return true
}

func BenchmarkCreateShortID(b *testing.B) {
	input := "https://example.com/very/long/path/to/article/with/many/segments/and/parameters?param1=value1&param2=value2"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.CreateShortID(input)
	}
}

func BenchmarkSanitizeText(b *testing.B) {
	input := "Длинный текст с различными символами: кириллица, латиница, цифры 123, эмодзи 😀, знаки препинания... и другие символы!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.SanitizeText(input)
	}
}
