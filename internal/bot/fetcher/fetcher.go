package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Article представляет одну новостную статью из ответа GNews.
type Article struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	URL         string    `json:"url"`
	Image       string    `json:"image"`
	PublishedAt time.Time `json:"publishedAt"`
	Source      Source    `json:"source"`
}

// Source представляет источник новости.
type Source struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// GNewsResponse представляет полный ответ от API GNews.
type GNewsResponse struct {
	TotalArticles int       `json:"totalArticles"`
	Articles      []Article `json:"articles"`
}

// Fetcher предоставляет методы для получения новостей из различных источников.
type Fetcher struct {
	GNewsAPIKey string
	NewsAPIKey  string
	HTTPClient  *http.Client
	LastAPIUsed string // Запоминаем последний использованный API
}

// NewFetcher создает новый экземпляр Fetcher.
func NewFetcher(gNewsAPIKey string, newsAPIKey string) *Fetcher {
	return &Fetcher{
		GNewsAPIKey: gNewsAPIKey,
		NewsAPIKey:  newsAPIKey,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second, // Устанавливаем таймаут для запросов
		},
		LastAPIUsed: "",
	}
}

// NewsAPIResponse представляет ответ от News API
type NewsAPIResponse struct {
	Status       string `json:"status"`
	TotalResults int    `json:"totalResults"`
	Articles     []struct {
		Source struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"source"`
		Author      string    `json:"author"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		URL         string    `json:"url"`
		URLToImage  string    `json:"urlToImage"`
		PublishedAt time.Time `json:"publishedAt"`
		Content     string    `json:"content"`
	} `json:"articles"`
}

// FetchNewsFromNewsAPI выполняет запрос к News API для получения новостей по теме
func (f *Fetcher) FetchNewsFromNewsAPI(topic string) ([]Article, error) {
	if f.NewsAPIKey == "" {
		return nil, fmt.Errorf("ключ News API не настроен")
	}

	log.Printf("Запрашиваю новости из News API по теме: '%s'", topic)

	// Не кодируем тему здесь, так как будем использовать модифицированный запрос

	// Расширяем запрос для получения большего количества результатов

	// Добавляем синонимы и исправления для популярных тем
	var searchQuery string
	switch topic {
	case "искусственный интелент":
		searchQuery = "искусственный интеллект"
	case "программирование":
		searchQuery = "программирование"
	case "политика":
		searchQuery = "политика"
	case "новости москвы":
		searchQuery = "москва новости"
	default:
		searchQuery = topic
	}

	// Кодируем запрос для URL
	query := url.QueryEscape(searchQuery)

	// Формируем URL для запроса с расширенными параметрами
	apiURL := fmt.Sprintf("https://newsapi.org/v2/everything?q=%s&language=ru&sortBy=publishedAt&pageSize=10&apiKey=%s", query, f.NewsAPIKey)
	log.Printf("Запрос к News API: %s", apiURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса к News API: %w", err)
	}

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса к News API: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Ответ от News API: статус %d %s", resp.StatusCode, resp.Status)

	if resp.StatusCode != http.StatusOK {
		// Читаем тело ответа для получения дополнительной информации об ошибке
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("News API вернул ошибку: %s, тело: %s", resp.Status, string(body))
	}

	var newsAPIResponse NewsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&newsAPIResponse); err != nil {
		return nil, fmt.Errorf("ошибка декодирования JSON от News API: %w", err)
	}

	log.Printf("Получено %d статей из News API по теме '%s'", newsAPIResponse.TotalResults, topic)

	// Преобразуем в наш формат статей
	articles := make([]Article, 0, len(newsAPIResponse.Articles))
	for _, a := range newsAPIResponse.Articles {
		articles = append(articles, Article{
			Title:       a.Title,
			Description: a.Description,
			Content:     a.Content,
			URL:         a.URL,
			Image:       a.URLToImage,
			PublishedAt: a.PublishedAt,
			Source: Source{
				Name: a.Source.Name,
				URL:  "", // News API не предоставляет URL источника
			},
		})
	}

	if len(articles) > 0 {
		log.Printf("Первая статья из News API: '%s', опубликована: %s",
			articles[0].Title,
			articles[0].PublishedAt.Format("2006-01-02 15:04:05"))
	}

	f.LastAPIUsed = "NewsAPI"
	return articles, nil
}

// FetchNewsFromGNews выполняет запрос к GNews API для получения новостей по теме
func (f *Fetcher) FetchNewsFromGNews(topic string) ([]Article, error) {
	if f.GNewsAPIKey == "" {
		return nil, fmt.Errorf("ключ GNews API не настроен")
	}

	log.Printf("Запрашиваю новости из GNews API по теме: '%s'", topic)

	// Расширяем запрос для получения большего количества результатов
	modifiedTopic := topic

	// Добавляем синонимы и исправления для популярных тем
	switch topic {
	case "искусственный интелент":
		modifiedTopic = "искусственный интеллект"
	}

	// Увеличиваем количество результатов до 20
	log.Printf("Использую модифицированную тему для GNews API: '%s'", modifiedTopic)

	// Кодируем модифицированную тему для URL
	encodedTopic := url.QueryEscape(modifiedTopic)

	// Формируем URL для запроса с увеличенным количеством результатов
	apiURL := fmt.Sprintf("https://gnews.io/api/v4/search?q=%s&country=ru&lang=ru&sortby=publishedAt&max=20&token=%s", encodedTopic, f.GNewsAPIKey)
	log.Printf("Запрос к GNews API: %s", apiURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса к GNews: %w", err)
	}

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса к GNews: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Ответ от GNews API: статус %d %s", resp.StatusCode, resp.Status)

	if resp.StatusCode != http.StatusOK {
		// Читаем тело ответа для получения дополнительной информации об ошибке
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GNews API вернул ошибку: %s, тело: %s", resp.Status, string(body))
	}

	var gnewsResponse GNewsResponse
	if err := json.NewDecoder(resp.Body).Decode(&gnewsResponse); err != nil {
		return nil, fmt.Errorf("ошибка декодирования JSON от GNews: %w", err)
	}

	log.Printf("Получено %d статей из GNews API по теме '%s'", len(gnewsResponse.Articles), topic)
	if len(gnewsResponse.Articles) > 0 {
		log.Printf("Первая статья из GNews: '%s', опубликована: %s",
			gnewsResponse.Articles[0].Title,
			gnewsResponse.Articles[0].PublishedAt.Format("2006-01-02 15:04:05"))
	}

	f.LastAPIUsed = "GNews"
	return gnewsResponse.Articles, nil
}

// FetchNews получает новости по теме из доступных источников.
func (f *Fetcher) FetchNews(topic string) ([]Article, error) {
	// Проверяем, не пустая ли тема
	if topic == "" {
		return nil, fmt.Errorf("тема не может быть пустой")
	}

	// Сначала пробуем GNews API
	articles, err := f.FetchNewsFromGNews(topic)
	if err == nil && len(articles) > 0 {
		return articles, nil
	}

	// Если GNews не удалось или нет результатов, пробуем News API
	if f.NewsAPIKey != "" {
		log.Printf("Не удалось получить новости из GNews API: %v. Пробую News API...", err)
		articles, err2 := f.FetchNewsFromNewsAPI(topic)
		if err2 == nil {
			return articles, nil
		}
		log.Printf("Не удалось получить новости из News API: %v", err2)
	}

	// Если не удалось получить новости ни из одного API
	if err != nil {
		return nil, fmt.Errorf("не удалось получить новости из доступных источников: %v", err)
	}

	return articles, nil
}
