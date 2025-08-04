package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/vladislavdragonenkov/news-telegram-bot/internal/bot/database"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(database.NewSQLiteDialector(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Автоматическая миграция для тестов
	err = db.AutoMigrate(&database.User{}, &database.Subscription{}, &database.FavoriteArticle{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestFavoriteArticleRepository_AddFavoriteArticle(t *testing.T) {
	db := setupTestDB(t)
	repo := database.NewFavoriteArticleRepository(db)
	ctx := context.Background()

	// Создаем тестового пользователя
	user := &database.User{
		TelegramID: 12345,
		Username:   "testuser",
		FirstName:  "Test",
		LastName:   "User",
	}
	err := db.Create(user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name         string
		userID       uint
		articleURL   string
		articleTitle string
		wantErr      bool
	}{
		{
			name:         "Valid article",
			userID:       user.ID,
			articleURL:   "https://example.com/article1",
			articleTitle: "Test Article 1",
			wantErr:      false,
		},
		{
			name:         "Different article same user",
			userID:       user.ID,
			articleURL:   "https://example.com/article2",
			articleTitle: "Test Article 2",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.AddFavoriteArticle(ctx, tt.userID, tt.articleURL, tt.articleTitle, "test-source", time.Now())
			if (err != nil) != tt.wantErr {
				t.Errorf("AddFavoriteArticle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFavoriteArticleRepository_IsFavoriteArticle(t *testing.T) {
	db := setupTestDB(t)
	repo := database.NewFavoriteArticleRepository(db)
	ctx := context.Background()

	// Создаем тестового пользователя
	user := &database.User{
		TelegramID: 12345,
		Username:   "testuser",
		FirstName:  "Test",
		LastName:   "User",
	}
	err := db.Create(user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Добавляем статью в избранное
	favoriteURL := "https://example.com/favorite"
	err = repo.AddFavoriteArticle(ctx, user.ID, favoriteURL, "Favorite Article", "test-source", time.Now())
	if err != nil {
		t.Fatalf("Failed to add favorite article: %v", err)
	}

	tests := []struct {
		name       string
		userID     uint
		articleURL string
		want       bool
		wantErr    bool
	}{
		{
			name:       "Check existing favorite",
			userID:     user.ID,
			articleURL: favoriteURL,
			want:       true,
			wantErr:    false,
		},
		{
			name:       "Check non-existing favorite",
			userID:     user.ID,
			articleURL: "https://example.com/notfavorite",
			want:       false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.IsFavoriteArticle(ctx, tt.userID, tt.articleURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsFavoriteArticle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsFavoriteArticle() = %v, want %v", got, tt.want)
			}
		})
	}
}
