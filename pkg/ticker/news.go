package ticker

import (
	"encoding/json"
	"fmt"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// newsAPIResponse represents the Yahoo Finance news API response structure.
type newsAPIResponse struct {
	Data struct {
		TickerStream struct {
			Stream []newsStreamItem `json:"stream"`
		} `json:"tickerStream"`
	} `json:"data"`
}

// newsStreamItem represents a single item in the news stream.
type newsStreamItem struct {
	ID      string `json:"id"`
	Content struct {
		ID           string `json:"id"`
		ContentType  string `json:"contentType"`
		Title        string `json:"title"`
		PubDate      string `json:"pubDate"`
		Thumbnail    *struct {
			Resolutions []models.ThumbnailResolution `json:"resolutions"`
		} `json:"thumbnail"`
		ClickThroughURL *struct {
			URL string `json:"url"`
		} `json:"clickThroughUrl"`
		Provider struct {
			DisplayName string `json:"displayName"`
		} `json:"provider"`
	} `json:"content"`
	Ad []interface{} `json:"ad"`
}

// News fetches news articles for the ticker.
//
// Parameters:
//   - count: Number of articles to fetch (default: 10)
//   - tab: Type of news to fetch (default: NewsTabNews)
//
// Example:
//
//	news, err := ticker.News(10, models.NewsTabNews)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, article := range news {
//	    fmt.Printf("%s: %s\n", article.Publisher, article.Title)
//	}
func (t *Ticker) News(count int, tab models.NewsTab) ([]models.NewsArticle, error) {
	// Check cache first
	t.mu.RLock()
	if t.newsCache != nil {
		cached := t.newsCache
		t.mu.RUnlock()
		return cached, nil
	}
	t.mu.RUnlock()

	// Set defaults
	if count <= 0 {
		count = 10
	}
	if tab == "" {
		tab = models.NewsTabNews
	}

	// Build request URL
	url := fmt.Sprintf("%s?queryRef=%s&serviceKey=ncp_fin",
		endpoints.NewsURL, tab.QueryRef())

	// Build request body
	payload := map[string]interface{}{
		"serviceConfig": map[string]interface{}{
			"snippetCount": count,
			"s":            []string{t.symbol},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal news request: %w", err)
	}

	// Make POST request with JSON body
	resp, err := t.client.PostJSON(url, nil, body)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch news: %w", err)
	}

	if resp.StatusCode == 429 {
		return nil, client.WrapRateLimitError()
	}
	if resp.StatusCode >= 400 {
		return nil, client.HTTPStatusToError(resp.StatusCode, resp.Body)
	}

	// Parse response
	var apiResp newsAPIResponse
	if err := json.Unmarshal([]byte(resp.Body), &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse news response: %w", err)
	}

	// Convert to NewsArticle slice, filtering out ads
	articles := make([]models.NewsArticle, 0, len(apiResp.Data.TickerStream.Stream))
	for _, item := range apiResp.Data.TickerStream.Stream {
		// Skip ads
		if len(item.Ad) > 0 {
			continue
		}

		article := models.NewsArticle{
			UUID:      item.Content.ID,
			Title:     item.Content.Title,
			Publisher: item.Content.Provider.DisplayName,
			Type:      item.Content.ContentType,
		}

		if item.Content.ClickThroughURL != nil {
			article.Link = item.Content.ClickThroughURL.URL
		}

		if item.Content.Thumbnail != nil {
			article.Thumbnail = &models.NewsThumbnail{
				Resolutions: item.Content.Thumbnail.Resolutions,
			}
		}

		articles = append(articles, article)
	}

	// Cache the results
	t.mu.Lock()
	t.newsCache = articles
	t.mu.Unlock()

	return articles, nil
}

// GetNews is an alias for News with default parameters.
// It fetches 10 news articles of type "news".
//
// Example:
//
//	news, err := ticker.GetNews()
func (t *Ticker) GetNews() ([]models.NewsArticle, error) {
	return t.News(10, models.NewsTabNews)
}
