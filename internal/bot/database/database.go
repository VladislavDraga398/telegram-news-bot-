package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// database реализует интерфейс Database и встраивает в себя все репозитории.
type database struct {
	UserRepository
	SubscriptionRepository
	SentArticleRepository
	FavoriteArticleRepository
	db *gorm.DB
}

// Ensure database implements Database interface.
var _ Database = (*database)(nil)

const (
	MaxTopicLength    = 255
	MaxUsernameLength = 64
	MaxNameLength     = 64
)

// User представляет пользователя бота.
type User struct {
	gorm.Model
	TelegramID                  int64  `gorm:"uniqueIndex;not null"`
	Username                    string `gorm:"size:64"`
	FirstName                   string `gorm:"size:64;not null"`
	LastName                    string `gorm:"size:64"`
	State                       string `gorm:"default:''"`
	NotificationIntervalMinutes uint   `gorm:"default:60"`
	LastNotifiedAt              *time.Time
	NewsLimit                   uint           `gorm:"default:5"` // Количество новостей для получения, по умолчанию 5
	Subscriptions               []Subscription `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// Subscription представляет подписку пользователя на тему.
type Subscription struct {
	gorm.Model
	UserID uint   `gorm:"index;not null"`
	Topic  string `gorm:"size:255;not null"`
}

// SentArticle отслеживает отправленные статьи.
type SentArticle struct {
	gorm.Model
	UserID      uint   `gorm:"not null;index"`
	ArticleHash string `gorm:"not null;index"`
	SentAt      time.Time
}

// FavoriteArticle представляет избранную новость пользователя.
type FavoriteArticle struct {
	gorm.Model
	UserID      uint      `gorm:"not null;index"`
	ArticleURL  string    `gorm:"not null;index"`
	Title       string    `gorm:"not null"`
	Source      string    `gorm:"not null"`
	PublishedAt time.Time `gorm:"not null"`
	AddedAt     time.Time `gorm:"not null"`
}

// New создает и инициализирует новый экземпляр базы данных.
func New(dbPath string) (Database, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Инициализация базы данных с использованием альтернативного драйвера SQLite без CGO
	db, err := gorm.Open(NewSQLiteDialector(dbPath), &gorm.Config{
		Logger:      newLogger,
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err = db.AutoMigrate(&User{}, &Subscription{}, &SentArticle{}, &FavoriteArticle{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database connection and migration successful.")

	return &database{
		UserRepository:            NewUserRepository(db),
		SubscriptionRepository:    NewSubscriptionRepository(db),
		SentArticleRepository:     NewSentArticleRepository(db),
		FavoriteArticleRepository: NewFavoriteArticleRepository(db),
		db:                        db,
	}, nil
}

// GetDB возвращает *gorm.DB.
func (d *database) GetDB() *gorm.DB {
	return d.db
}

// Close закрывает соединение с базой данных.
func (d *database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB for closing: %w", err)
	}
	return sqlDB.Close()
}

// userRepository реализует UserRepository.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository создает новый репозиторий пользователей.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindOrCreateUser(ctx context.Context, telegramID int64, username, firstName, lastName string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where(User{TelegramID: telegramID}).FirstOrInit(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to find or init user: %w", err)
	}

	if user.ID == 0 {
		user.Username = username
		user.FirstName = firstName
		user.LastName = lastName
		if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	return &user, nil
}

func (r *userRepository) UpdateUserNotificationInterval(ctx context.Context, userID uint, intervalMinutes uint) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("notification_interval_minutes", intervalMinutes).Error
}

func (r *userRepository) GetAllUsers(ctx context.Context) ([]User, error) {
	var users []User
	if err := r.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	return users, nil
}

func (r *userRepository) UpdateUserLastNotifiedAt(ctx context.Context, userID uint, notifyTime time.Time) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("last_notified_at", notifyTime).Error
}

func (r *userRepository) SetUserState(ctx context.Context, userID uint, state string) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("state", state).Error
}

func (r *userRepository) UpdateUserNewsLimit(ctx context.Context, userID uint, newsLimit uint) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("news_limit", newsLimit).Error
}

func (r *userRepository) GetUserState(ctx context.Context, userID uint) (string, error) {
	var user User
	if err := r.db.WithContext(ctx).Select("state").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return user.State, nil
}

// subscriptionRepository реализует SubscriptionRepository.
type subscriptionRepository struct {
	db *gorm.DB
}

// NewSubscriptionRepository создает новый репозиторий подписок.
func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) AddSubscription(ctx context.Context, userID uint, topic string) error {
	subscription := Subscription{
		UserID: userID,
		Topic:  strings.ToLower(topic),
	}

	var count int64
	r.db.WithContext(ctx).Model(&Subscription{}).Where("user_id = ? AND topic = ?", userID, subscription.Topic).Count(&count)
	if count > 0 {
		return errors.New("подписка на эту тему уже существует")
	}

	return r.db.WithContext(ctx).Create(&subscription).Error
}

func (r *subscriptionRepository) RemoveSubscription(ctx context.Context, userID uint, topic string) error {
	tx := r.db.WithContext(ctx).Where("user_id = ? AND topic = ?", userID, strings.ToLower(topic)).Delete(&Subscription{})
	if tx.Error != nil {
		return fmt.Errorf("failed to remove subscription: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return errors.New("subscription not found")
	}
	return nil
}

func (r *subscriptionRepository) GetUserSubscriptions(ctx context.Context, userID uint) ([]string, error) {
	var topics []string
	err := r.db.WithContext(ctx).Model(&Subscription{}).Where("user_id = ?", userID).Pluck("topic", &topics).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscriptions: %w", err)
	}
	return topics, nil
}

func (r *subscriptionRepository) GetAllUniqueTopics(ctx context.Context) ([]string, error) {
	var topics []string
	err := r.db.WithContext(ctx).Model(&Subscription{}).Distinct().Pluck("topic", &topics).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get unique topics: %w", err)
	}
	return topics, nil
}

func (r *subscriptionRepository) GetSubscribersForTopic(ctx context.Context, topic string) ([]int64, error) {
	var userIDs []int64
	err := r.db.WithContext(ctx).Model(&Subscription{}).
		Joins("join users on users.id = subscriptions.user_id").
		Where("subscriptions.topic = ?", topic).
		Pluck("users.telegram_id", &userIDs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get subscribers for topic %s: %w", topic, err)
	}
	return userIDs, nil
}

// sentArticleRepository реализует SentArticleRepository.
type sentArticleRepository struct {
	db *gorm.DB
}

// NewSentArticleRepository создает новый репозиторий.
func NewSentArticleRepository(db *gorm.DB) SentArticleRepository {
	return &sentArticleRepository{db: db}
}

func (r *sentArticleRepository) IsArticleSent(ctx context.Context, userID uint, articleHash string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SentArticle{}).Where("user_id = ? AND article_hash = ?", userID, articleHash).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check if article was sent: %w", err)
	}
	return count > 0, nil
}

// ResetSentArticlesHistory удаляет всю историю отправленных статей для указанного пользователя
func (r *sentArticleRepository) ResetSentArticlesHistory(ctx context.Context, userID uint) error {
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&SentArticle{}).Error
	if err != nil {
		return fmt.Errorf("failed to reset sent articles history: %w", err)
	}
	return nil
}

func (r *sentArticleRepository) MarkArticleAsSent(ctx context.Context, userID uint, articleHash string) error {
	sentArticle := SentArticle{
		UserID:      userID,
		ArticleHash: articleHash,
		SentAt:      time.Now(),
	}
	return r.db.WithContext(ctx).Create(&sentArticle).Error
}

// MigrateSubscriptionsToLower конвертирует все темы подписок в нижний регистр для обеспечения
// регистронезависимого поиска и сравнения.
func MigrateSubscriptionsToLower(db *gorm.DB) error {
	log.Println("Запуск миграции подписок к нижнему регистру...")

	var subscriptions []Subscription
	if err := db.Find(&subscriptions).Error; err != nil {
		return fmt.Errorf("failed to fetch subscriptions: %w", err)
	}

	for _, sub := range subscriptions {
		lowerTopic := strings.ToLower(sub.Topic)
		if sub.Topic != lowerTopic {
			log.Printf("Миграция подписки ID %d: '%s' -> '%s'", sub.ID, sub.Topic, lowerTopic)
			if err := db.Model(&sub).Update("topic", lowerTopic).Error; err != nil {
				return fmt.Errorf("failed to update subscription %d: %w", sub.ID, err)
			}
		}
	}

	log.Println("Миграция подписок завершена успешно.")
	return nil
}
