package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// NewsArticle represents a single news article from Yahoo Finance.
type NewsArticle struct {
	// UUID is the unique identifier for the article.
	UUID string `json:"uuid"`

	// Title is the headline of the article.
	Title string `json:"title"`

	// Publisher is the source of the article.
	Publisher string `json:"publisher"`

	// Link is the URL to the full article. It is Yahoo's click-through URL,
	// falling back to CanonicalURL when Yahoo does not provide one.
	Link string `json:"link"`

	// PublishTime is the Unix timestamp when the article was published.
	// It is 0 when Yahoo does not report a parseable publish time.
	PublishTime int64 `json:"providerPublishTime"`

	// Type is the content type (e.g., "STORY", "VIDEO", "OUTLINK").
	Type string `json:"type"`

	// Thumbnail contains image URLs for the article.
	Thumbnail *NewsThumbnail `json:"thumbnail,omitempty"`

	// RelatedTickers lists ticker symbols related to this article.
	RelatedTickers []string `json:"relatedTickers,omitempty"`

	// Summary is Yahoo's short summary of the article, when available.
	Summary string `json:"summary,omitempty"`

	// Description is the article description or teaser, when available.
	Description string `json:"description,omitempty"`

	// CanonicalURL is the publisher's canonical article URL, when available.
	CanonicalURL string `json:"canonicalUrl,omitempty"`

	// PreviewURL is Yahoo's preview URL for the article, when available.
	PreviewURL string `json:"previewUrl,omitempty"`

	// PublisherURL is the publisher's home page URL, when available.
	PublisherURL string `json:"publisherUrl,omitempty"`

	// DisplayTime is the Unix timestamp Yahoo displays for the article.
	// It is 0 when Yahoo does not report a parseable display time.
	DisplayTime int64 `json:"displayTime,omitempty"`

	// IsHosted reports whether the article is hosted on Yahoo Finance.
	IsHosted bool `json:"isHosted,omitempty"`

	// IsPremium reports whether the article is Yahoo Finance premium news.
	IsPremium bool `json:"isPremium,omitempty"`
}

// NewsThumbnail contains thumbnail image information for a news article.
// It reuses ThumbnailResolution from the search package.
type NewsThumbnail struct {
	// OriginalURL is the URL of the original (unresized) image.
	OriginalURL string `json:"originalUrl,omitempty"`

	// OriginalWidth is the width of the original image in pixels.
	OriginalWidth int `json:"originalWidth,omitempty"`

	// OriginalHeight is the height of the original image in pixels.
	OriginalHeight int `json:"originalHeight,omitempty"`

	// Resolutions contains different size versions of the thumbnail.
	// The first entry is the original image (tag "original"), followed by
	// resized versions tagged with Yahoo's transform label (e.g. "170x128").
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

// ErrInvalidNewsTab is returned (wrapped) when a NewsTab is not one of
// NewsTabAll, NewsTabNews or NewsTabPressReleases.
var ErrInvalidNewsTab = errors.New("invalid news tab")

// String returns the string representation of the NewsTab.
func (t NewsTab) String() string {
	return string(t)
}

// ListName returns the Yahoo Finance news list name for this tab, as used in
// the ticker news list slug: "news", "all" or "pressrelease".
//
// Matching is case-insensitive (as in Python yfinance), and the
// empty tab maps to the default "news" list. Any other value returns an error
// wrapping ErrInvalidNewsTab.
func (t NewsTab) ListName() (string, error) {
	switch NewsTab(strings.ToLower(string(t))) {
	case "", NewsTabNews:
		return "news", nil
	case NewsTabAll:
		return "all", nil
	case NewsTabPressReleases:
		return "pressrelease", nil
	default:
		return "", fmt.Errorf("%w %q: choose from %q, %q, %q",
			ErrInvalidNewsTab, string(t), NewsTabNews, NewsTabAll, NewsTabPressReleases)
	}
}

// Slug returns the Yahoo Finance news list slug for this tab and locale, for
// example "list=finance-US-en-US-ticker-news".
//
// It returns an error wrapping ErrInvalidNewsTab for an unknown tab.
func (t NewsTab) Slug(lang, region string) (string, error) {
	listName, err := t.ListName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("list=finance-%s-%s-ticker-%s", region, lang, listName), nil
}

// QueryRef returns the Yahoo Finance API query reference for this tab.
//
// Deprecated: QueryRef belongs to Yahoo's retired /xhr/ncp news endpoint and
// is no longer used by Ticker.News. Use ListName or Slug instead.
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
