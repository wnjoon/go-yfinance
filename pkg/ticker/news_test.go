package ticker

import (
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNewsTabQueryRef(t *testing.T) {
	tests := []struct {
		tab      models.NewsTab
		expected string
	}{
		{models.NewsTabAll, "newsAll"},
		{models.NewsTabNews, "latestNews"},
		{models.NewsTabPressReleases, "pressRelease"},
		{models.NewsTab(""), "latestNews"}, // Default
	}

	for _, tt := range tests {
		t.Run(string(tt.tab), func(t *testing.T) {
			got := tt.tab.QueryRef()
			if got != tt.expected {
				t.Errorf("NewsTab(%q).QueryRef() = %s, want %s", tt.tab, got, tt.expected)
			}
		})
	}
}

func TestNewsArticlePublishedAt(t *testing.T) {
	timestamp := int64(1705329000) // 2024-01-15 14:30:00 UTC
	article := &models.NewsArticle{
		PublishTime: timestamp,
	}

	publishedAt := article.PublishedAt()
	expected := time.Unix(timestamp, 0)

	if !publishedAt.Equal(expected) {
		t.Errorf("PublishedAt() = %v, want %v", publishedAt, expected)
	}
}

func TestNewsTabString(t *testing.T) {
	tests := []struct {
		tab      models.NewsTab
		expected string
	}{
		{models.NewsTabAll, "all"},
		{models.NewsTabNews, "news"},
		{models.NewsTabPressReleases, "press releases"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tab), func(t *testing.T) {
			got := tt.tab.String()
			if got != tt.expected {
				t.Errorf("NewsTab.String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

// Integration test - requires network access
// func TestNewsIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	ticker, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer ticker.Close()
//
// 	news, err := ticker.News(5, models.NewsTabNews)
// 	if err != nil {
// 		t.Fatalf("Failed to get news: %v", err)
// 	}
//
// 	t.Logf("Got %d news articles", len(news))
// 	for _, article := range news {
// 		t.Logf("  - %s: %s", article.Publisher, article.Title)
// 	}
// }
