package ticker

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Valuation returns the key-statistics valuation measures table.
//
// This mirrors Python yfinance's ticker.valuation property added in v1.3.0.
func (t *Ticker) Valuation() (*models.ValuationMeasures, error) {
	t.mu.RLock()
	if t.valuationCache != nil {
		cached := t.valuationCache
		t.mu.RUnlock()
		return cached, nil
	}
	t.mu.RUnlock()

	statsURL := fmt.Sprintf("%s/quote/%s/key-statistics", endpoints.RootURL, url.PathEscape(t.symbol))
	resp, err := t.client.Get(statsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch key-statistics page: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, client.HTTPStatusToError(resp.StatusCode, resp.Body)
	}

	measures, err := parseValuationMeasuresHTML(resp.Body)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	t.valuationCache = measures
	t.mu.Unlock()

	return measures, nil
}

// ValuationMeasures is an alias for Valuation.
func (t *Ticker) ValuationMeasures() (*models.ValuationMeasures, error) {
	return t.Valuation()
}

func parseValuationMeasuresHTML(body string) (*models.ValuationMeasures, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse key-statistics HTML: %w", err)
	}

	table := firstElement(doc, "table")
	if table == nil {
		return &models.ValuationMeasures{}, nil
	}

	rawRows := tableRows(table)
	if len(rawRows) == 0 {
		return &models.ValuationMeasures{}, nil
	}

	headers := rawRows[0]
	if len(headers) <= 1 {
		return &models.ValuationMeasures{}, nil
	}

	columns := append([]string{}, headers[1:]...)
	rows := make([]models.ValuationMeasureRow, 0, len(rawRows)-1)
	for _, rawRow := range rawRows[1:] {
		if len(rawRow) == 0 || rawRow[0] == "" {
			continue
		}
		row := models.ValuationMeasureRow{
			Name:   rawRow[0],
			Values: make(map[string]string, len(columns)),
		}
		for i, column := range columns {
			cellIndex := i + 1
			if cellIndex < len(rawRow) {
				row.Values[column] = rawRow[cellIndex]
			} else {
				row.Values[column] = ""
			}
		}
		rows = append(rows, row)
	}

	return &models.ValuationMeasures{Columns: columns, Rows: rows}, nil
}

func firstElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := firstElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func tableRows(table *html.Node) [][]string {
	var rows [][]string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			rows = append(rows, tableCells(n))
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(table)
	return rows
}

func tableCells(row *html.Node) []string {
	cells := []string{}
	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && (child.Data == "th" || child.Data == "td") {
			cells = append(cells, strings.TrimSpace(nodeText(child)))
		}
	}
	return cells
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
