package models

import "time"

// NewsArticle represents a single news article from Yahoo Finance.
type NewsArticle struct {
	// UUID is the unique identifier for the article.
	UUID string `json:"uuid"`

	// Title is the headline of the article.
	Title string `json:"title"`

	// Publisher is the source of the article.
	Publisher string `json:"publisher"`

	// Link is the URL to the full article.
	Link string `json:"link"`

	// PublishTime is the Unix timestamp when the article was published.
	PublishTime int64 `json:"providerPublishTime"`

	// Type is the content type (e.g., "STORY", "VIDEO").
	Type string `json:"type"`

	// Thumbnail contains image URLs for the article.
	Thumbnail *NewsThumbnail `json:"thumbnail,omitempty"`

	// RelatedTickers lists ticker symbols related to this article.
	RelatedTickers []string `json:"relatedTickers,omitempty"`
}

// NewsThumbnail contains thumbnail image information for a news article.
// It reuses ThumbnailResolution from the search package.
type NewsThumbnail struct {
	// Resolutions contains different size versions of the thumbnail.
	Resolutions []ThumbnailResolution `json:"resolutions,omitempty"`
}

// PublishedAt returns the publish time as a time.Time value.
func (n *NewsArticle) PublishedAt() time.Time {
	return time.Unix(n.PublishTime, 0)
}

// NewsTab represents the type of news to fetch.
type NewsTab string

const (
	// NewsTabAll fetches all news including articles and press releases.
	NewsTabAll NewsTab = "all"

	// NewsTabNews fetches only news articles (default).
	NewsTabNews NewsTab = "news"

	// NewsTabPressReleases fetches only press releases.
	NewsTabPressReleases NewsTab = "press releases"
)

// String returns the string representation of the NewsTab.
func (t NewsTab) String() string {
	return string(t)
}

// QueryRef returns the Yahoo Finance API query reference for this tab.
func (t NewsTab) QueryRef() string {
	switch t {
	case NewsTabAll:
		return "newsAll"
	case NewsTabPressReleases:
		return "pressRelease"
	default:
		return "latestNews"
	}
}

// NewsParams contains parameters for fetching news articles.
type NewsParams struct {
	// Count is the number of articles to fetch (default: 10).
	Count int

	// Tab specifies the type of news to fetch.
	Tab NewsTab
}
