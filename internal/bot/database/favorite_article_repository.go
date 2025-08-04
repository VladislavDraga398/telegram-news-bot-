package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// favoriteArticleRepository реализует интерфейс FavoriteArticleRepository.
type favoriteArticleRepository struct {
	db *gorm.DB
}

// NewFavoriteArticleRepository создает новый экземпляр репозитория избранных статей.
func NewFavoriteArticleRepository(db *gorm.DB) FavoriteArticleRepository {
	return &favoriteArticleRepository{db: db}
}

// AddFavoriteArticle добавляет статью в избранное пользователя.
func (r *favoriteArticleRepository) AddFavoriteArticle(ctx context.Context, userID uint, articleURL string, title string, source string, publishedAt time.Time) error {
	// Проверяем, не добавлена ли уже эта статья в избранное
	var count int64
	if err := r.db.WithContext(ctx).Model(&FavoriteArticle{}).
		Where("user_id = ? AND article_url = ?", userID, articleURL).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check if article is already in favorites: %w", err)
	}

	if count > 0 {
		return errors.New("article is already in favorites")
	}

	// Добавляем статью в избранное
	favoriteArticle := FavoriteArticle{
		UserID:      userID,
		ArticleURL:  articleURL,
		Title:       title,
		Source:      source,
		PublishedAt: publishedAt,
		AddedAt:     time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&favoriteArticle).Error; err != nil {
		return fmt.Errorf("failed to add article to favorites: %w", err)
	}

	return nil
}

// RemoveFavoriteArticle удаляет статью из избранного пользователя.
func (r *favoriteArticleRepository) RemoveFavoriteArticle(ctx context.Context, userID uint, articleURL string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND article_url = ?", userID, articleURL).
		Delete(&FavoriteArticle{})

	if result.Error != nil {
		return fmt.Errorf("failed to remove article from favorites: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("article not found in favorites")
	}

	return nil
}

// GetUserFavoriteArticles возвращает список избранных статей пользователя.
func (r *favoriteArticleRepository) GetUserFavoriteArticles(ctx context.Context, userID uint) ([]FavoriteArticle, error) {
	var favoriteArticles []FavoriteArticle
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("added_at DESC").
		Find(&favoriteArticles).Error; err != nil {
		return nil, fmt.Errorf("failed to get user favorite articles: %w", err)
	}

	return favoriteArticles, nil
}

// IsFavoriteArticle проверяет, добавлена ли статья в избранное пользователя.
func (r *favoriteArticleRepository) IsFavoriteArticle(ctx context.Context, userID uint, articleURL string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&FavoriteArticle{}).
		Where("user_id = ? AND article_url = ?", userID, articleURL).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check if article is in favorites: %w", err)
	}

	return count > 0, nil
}
