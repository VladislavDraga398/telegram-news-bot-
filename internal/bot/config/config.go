package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config хранит все конфигурационные параметры для бота.
type Config struct {
	Token       string
	GNewsAPIKey string
	NewsAPIKey  string
	DBPath      string
	Mode        string
	WebhookURL  string
	Port        string
	TLSCertPath string
	TLSKeyPath  string
}

// Load загружает конфигурацию из .env файла и флагов командной строки.
// Флаги командной строки имеют приоритет над .env файлом.
func Load() (*Config, error) {
	// Загружаем .env файл, но не считаем ошибкой, если его нет
	_ = godotenv.Load()

	var cfg Config

	// Получаем значения из переменных окружения как значения по умолчанию
	defaultToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	defaultGNewsAPIKey := os.Getenv("GNEWS_API_KEY")
	defaultNewsAPIKey := os.Getenv("NEWS_API_KEY")
	defaultDBPath := "data/bot.db"
	defaultMode := "polling"

	// Определяем флаги командной строки
	flag.StringVar(&cfg.Token, "token", defaultToken, "Telegram Bot Token")
	flag.StringVar(&cfg.GNewsAPIKey, "gnews-api-key", defaultGNewsAPIKey, "GNews API Key")
	flag.StringVar(&cfg.NewsAPIKey, "news-api-key", defaultNewsAPIKey, "News API Key")
	flag.StringVar(&cfg.DBPath, "db-path", defaultDBPath, "Path to SQLite database file")
	flag.StringVar(&cfg.Mode, "mode", defaultMode, "Bot mode (polling or webhook)")
	flag.StringVar(&cfg.WebhookURL, "webhook-url", "", "Webhook URL for webhook mode")
	flag.StringVar(&cfg.Port, "port", "8443", "Port for webhook server")
	flag.StringVar(&cfg.TLSCertPath, "tls-cert-path", "", "Path to TLS certificate file")
	flag.StringVar(&cfg.TLSKeyPath, "tls-key-path", "", "Path to TLS key file")

	flag.Parse()

	// Если токен все еще пуст после всех проверок, это ошибка
	if cfg.Token == "" {
		return nil, fmt.Errorf("токен бота не указан. Укажите его через флаг -token или в .env файле")
	}

	return &cfg, nil
}
