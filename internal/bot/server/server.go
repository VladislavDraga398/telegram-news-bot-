package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/handlers"
)

// Server представляет HTTP-сервер для обработки вебхуков
type Server struct {
	bot     *tgbotapi.BotAPI
	handler *handlers.Handler
	config  Config
	server  *http.Server
}

// Config содержит конфигурацию сервера
type Config struct {
	Port        string
	WebhookURL  string
	TLSCertPath string
	TLSKeyPath  string
}

// New создает новый экземпляр сервера
func New(bot *tgbotapi.BotAPI, handler *handlers.Handler, cfg Config) *Server {
	return &Server{
		bot:     bot,
		handler: handler,
		config:  cfg,
	}
}

// Start запускает сервер
func (s *Server) Start(ctx context.Context) error {
	// Удаляем предыдущий вебхук, если он был
	if _, err := s.bot.Request(tgbotapi.DeleteWebhookConfig{}); err != nil {
		log.Printf("Не удалось удалить предыдущий вебхук: %v", err)
	}

	// Настраиваем вебхук
	if err := s.setupWebhook(); err != nil {
		return fmt.Errorf("ошибка настройки вебхука: %w", err)
	}

	// Настраиваем HTTP-сервер
	s.setupHTTPServer()

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Printf("Запуск вебхук-сервера на порту %s...", s.config.Port)

		var err error
		if s.config.TLSCertPath != "" && s.config.TLSKeyPath != "" {
			err = s.server.ListenAndServeTLS(s.config.TLSCertPath, s.config.TLSKeyPath)
		} else {
			err = s.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка вебхук-сервера: %v", err)
		}
	}()

	// Ожидаем сигнала завершения
	<-ctx.Done()
	log.Println("Получен сигнал завершения, останавливаем сервер...")

	// Плавно завершаем работу сервера
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при завершении работы сервера: %v", err)
	}

	// Удаляем вебхук при завершении
	if _, err := s.bot.Request(tgbotapi.DeleteWebhookConfig{}); err != nil {
		log.Printf("Ошибка при удалении вебхука: %v", err)
	}

	return nil
}

func (s *Server) setupWebhook() error {
	// Создаем URL для вебхука
	webhookURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(s.config.WebhookURL, "/"), s.bot.Token)

	// Настраиваем вебхук в зависимости от наличия сертификата
	if s.config.TLSCertPath != "" && s.config.TLSKeyPath != "" {
		// Проверяем сертификат
		if _, certErr := tls.LoadX509KeyPair(s.config.TLSCertPath, s.config.TLSKeyPath); certErr != nil {
			return fmt.Errorf("ошибка загрузки сертификата: %v", certErr)
		}

		// Устанавливаем вебхук с сертификатом
		webhookCfg, err := tgbotapi.NewWebhookWithCert(webhookURL, tgbotapi.FilePath(s.config.TLSCertPath))
		if err != nil {
			return fmt.Errorf("ошибка при создании конфигурации вебхука с сертификатом: %v", err)
		}
		if _, reqErr := s.bot.Request(webhookCfg); reqErr != nil {
			return fmt.Errorf("ошибка при установке вебхука с сертификатом: %v", reqErr)
		}
		log.Printf("Используется TLS сертификат: %s", s.config.TLSCertPath)
	} else {
		// Устанавливаем вебхук без сертификата (для локальной разработки)
		webhookCfg, err := tgbotapi.NewWebhook(webhookURL)
		if err != nil {
			return fmt.Errorf("ошибка при создании конфигурации вебхука: %v", err)
		}
		if _, reqErr := s.bot.Request(webhookCfg); reqErr != nil {
			return fmt.Errorf("ошибка при установке вебхука: %v", reqErr)
		}
	}

	// Получаем информацию о вебхуке
	info, err := s.bot.GetWebhookInfo()
	if err != nil {
		log.Printf("Ошибка при получении информации о вебхуке: %v", err)
	} else {
		log.Printf("Вебхук установлен: %+v", info)
	}

	return nil
}

func (s *Server) setupHTTPServer() {
	// Создаем маршрутизатор
	mux := http.NewServeMux()

	// Обработчик обновлений от Telegram
	mux.HandleFunc("/"+s.bot.Token, func(w http.ResponseWriter, r *http.Request) {
		update, err := s.bot.HandleUpdate(r)
		if err != nil {
			log.Printf("Ошибка при обработке обновления: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		go func(u tgbotapi.Update) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Паника при обработке обновления: %v", r)
				}
			}()

			s.handler.HandleUpdate(u)
		}(*update)

		w.WriteHeader(http.StatusOK)
	})

	// Эндпоинт для проверки работоспособности
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Printf("Ошибка при записи ответа: %v", err)
		}
	})

	s.server = &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: mux,
	}
}
