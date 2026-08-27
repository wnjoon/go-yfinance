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

func TestNewsCacheResultsAreDeepCopies(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	tkr.newsCache = []models.NewsArticle{{
		Title:          "original",
		RelatedTickers: []string{"AAPL"},
		Thumbnail: &models.NewsThumbnail{Resolutions: []models.ThumbnailResolution{{
			URL: "https://example.test/original.jpg",
		}}},
	}}
	tkr.newsCacheCount = 10
	tkr.newsCacheTab = models.NewsTabNews

	first, err := tkr.News(10, models.NewsTabNews)
	if err != nil {
		t.Fatalf("News() error: %v", err)
	}
	first[0].Title = "mutated"
	first[0].RelatedTickers[0] = "MSFT"
	first[0].Thumbnail.Resolutions[0].URL = "https://example.test/mutated.jpg"

	second, err := tkr.News(10, models.NewsTabNews)
	if err != nil {
		t.Fatalf("second News() error: %v", err)
	}
	if second[0].Title != "original" || second[0].RelatedTickers[0] != "AAPL" {
		t.Fatalf("news cache mutated: %+v", second[0])
	}
	if got := second[0].Thumbnail.Resolutions[0].URL; got != "https://example.test/original.jpg" {
		t.Fatalf("nested resolution cache mutated: %q", got)
	}
}

func TestNewsCacheKeyIncludesCountAndTab(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	tkr.newsCache = []models.NewsArticle{{Title: "cached"}}
	tkr.newsCacheCount = 5
	tkr.newsCacheTab = models.NewsTabNews

	tkr.mu.RLock()
	matching := tkr.newsCacheMatches(5, models.NewsTabNews)
	wrongCount := tkr.newsCacheMatches(50, models.NewsTabNews)
	wrongTab := tkr.newsCacheMatches(5, models.NewsTabPressReleases)
	tkr.mu.RUnlock()
	if !matching || wrongCount || wrongTab {
		t.Fatalf("cache key comparison: matching=%v wrongCount=%v wrongTab=%v", matching, wrongCount, wrongTab)
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
