package preprocess

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/soli0222/diary-cli/internal/models"
)

var urlPattern = regexp.MustCompile(`https?://[^\s]+`)

// SummalyClient fetches link metadata via a Summaly-compatible endpoint.
type SummalyClient struct {
	endpoint   string
	httpClient *http.Client
}

// SummalyResponse is the response body from the Summaly service.
type SummalyResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Sitename    string `json:"sitename"`
	URL         string `json:"url"`
}

// NewSummalyClientWithEndpoint creates a Summaly client with a custom endpoint.
func NewSummalyClientWithEndpoint(endpoint string) *SummalyClient {
	endpoint = strings.TrimSpace(strings.TrimRight(endpoint, "/"))
	if endpoint == "" {
		return nil
	}

	return &SummalyClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// EnrichNotesWithSummaly appends link summaries to note text for URLs in each note.
// Spotify links are ignored.
func EnrichNotesWithSummaly(notes []models.Note, client *SummalyClient) []models.Note {
	if client == nil {
		return notes
	}

	enriched := make([]models.Note, len(notes))
	copy(enriched, notes)

	cache := make(map[string]string)

	for i := range enriched {
		if enriched[i].Text == nil || *enriched[i].Text == "" {
			continue
		}

		text := *enriched[i].Text
		urls := ExtractSummalyTargets(text)
		if len(urls) == 0 {
			continue
		}

		var lines []string
		for _, rawURL := range urls {
			if summary, ok := cache[rawURL]; ok {
				if summary != "" {
					lines = append(lines, summary)
				}
				continue
			}

			resp, err := client.Fetch(rawURL)
			if err != nil {
				cache[rawURL] = ""
				continue
			}

			summary := formatSummalyLine(resp)
			cache[rawURL] = summary
			if summary != "" {
				lines = append(lines, summary)
			}
		}

		if len(lines) == 0 {
			continue
		}

		newText := text + "\n\n[link summary]\n" + strings.Join(lines, "\n")
		enriched[i].Text = &newText
	}

	return enriched
}

// ExtractSummalyTargets extracts unique non-Spotify URLs from text.
func ExtractSummalyTargets(text string) []string {
	matches := urlPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var urls []string
	for _, m := range matches {
		normalized := normalizeURLToken(m)
		if normalized == "" || IsSpotifyURL(normalized) {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		urls = append(urls, normalized)
	}

	return urls
}

// IsSpotifyURL returns true if the URL host is spotify.
func IsSpotifyURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "spotify.com" || host == "www.spotify.com" || host == "open.spotify.com"
}

// Fetch retrieves metadata for a URL from Summaly.
func (c *SummalyClient) Fetch(rawURL string) (*SummalyResponse, error) {
	q := url.QueryEscape(rawURL)
	endpoint := c.endpoint + "?url=" + q

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("summaly returned status %d", resp.StatusCode)
	}

	var result SummalyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func normalizeURLToken(token string) string {
	trimmed := strings.TrimSpace(token)
	trimmed = strings.TrimRight(trimmed, ".,!?;:)]}＞」』\"'")

	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}

	return trimmed
}

func formatSummalyLine(resp *SummalyResponse) string {
	if resp == nil {
		return ""
	}

	title := strings.TrimSpace(resp.Title)
	desc := truncate(strings.TrimSpace(strings.ReplaceAll(resp.Description, "\n", " ")), 140)
	site := strings.TrimSpace(resp.Sitename)
	link := strings.TrimSpace(resp.URL)

	if title == "" && desc == "" {
		return ""
	}

	parts := []string{"- "}
	if title != "" {
		parts = append(parts, title)
	}
	if site != "" {
		parts = append(parts, " (", site, ")")
	}
	if desc != "" {
		parts = append(parts, ": ", desc)
	}
	if link != "" {
		parts = append(parts, " [", link, "]")
	}

	return strings.Join(parts, "")
}

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
