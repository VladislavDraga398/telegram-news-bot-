package database

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Database определяет основной интерфейс для работы с базой данных.
// Он объединяет в себе все репозитории.
type Database interface {
	UserRepository
	SubscriptionRepository
	SentArticleRepository
	FavoriteArticleRepository
	Close() error
	GetDB() *gorm.DB
}

// UserRepository определяет операции для работы с пользователями.
type UserRepository interface {
	FindOrCreateUser(ctx context.Context, telegramID int64, username, firstName, lastName string) (*User, error)
	GetAllUsers(ctx context.Context) ([]User, error)
	SetUserState(ctx context.Context, userID uint, state string) error
	GetUserState(ctx context.Context, userID uint) (string, error)
	UpdateUserLastNotifiedAt(ctx context.Context, userID uint, notifyTime time.Time) error
	UpdateUserNotificationInterval(ctx context.Context, userID uint, intervalMinutes uint) error
	UpdateUserNewsLimit(ctx context.Context, userID uint, newsLimit uint) error
}

// SubscriptionRepository определяет операции для работы с подписками.
type SubscriptionRepository interface {
	AddSubscription(ctx context.Context, userID uint, topic string) error
	RemoveSubscription(ctx context.Context, userID uint, topic string) error
	GetUserSubscriptions(ctx context.Context, userID uint) ([]string, error)
	GetAllUniqueTopics(ctx context.Context) ([]string, error)
	GetSubscribersForTopic(ctx context.Context, topic string) ([]int64, error)
}

// SentArticleRepository определяет операции для отслеживания отправленных статей.
type SentArticleRepository interface {
	IsArticleSent(ctx context.Context, userID uint, articleHash string) (bool, error)
	MarkArticleAsSent(ctx context.Context, userID uint, articleHash string) error
	ResetSentArticlesHistory(ctx context.Context, userID uint) error
}

// FavoriteArticleRepository определяет операции для работы с избранными статьями.
type FavoriteArticleRepository interface {
	AddFavoriteArticle(ctx context.Context, userID uint, articleURL string, title string, source string, publishedAt time.Time) error
	RemoveFavoriteArticle(ctx context.Context, userID uint, articleURL string) error
	GetUserFavoriteArticles(ctx context.Context, userID uint) ([]FavoriteArticle, error)
	IsFavoriteArticle(ctx context.Context, userID uint, articleURL string) (bool, error)
}
