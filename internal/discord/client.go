package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	webhookURL string
	httpClient *http.Client
}

const maxDescriptionLength = 4096

type webhookMessage struct {
	Username string         `json:"username,omitempty"`
	Embeds   []discordEmbed `json:"embeds,omitempty"`
}

type discordEmbed struct {
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Fields      []discordField `json:"fields,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

func NewClient(webhookURL string) *Client {
	return &Client{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) PostSummary(date string, noteCount int, title, summary string) error {
	payload := webhookMessage{
		Username: "diary-cli",
		Embeds: []discordEmbed{{
			Title:       fmt.Sprintf("%s のMisskeyサマリー", date),
			Description: truncate(summary, maxDescriptionLength),
			Color:       0x86b300,
			Timestamp:   time.Now().Format(time.RFC3339),
			Fields: []discordField{
				{Name: "タイトル", Value: title},
				{Name: "ノート数", Value: fmt.Sprintf("%d", noteCount), Inline: true},
			},
		}},
	}
	return c.send(payload)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func (c *Client) send(message webhookMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API error: %d - %s", resp.StatusCode, string(responseBody))
	}
	return nil
}
