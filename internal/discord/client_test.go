package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPostSummaryBuildsWebhookPayload(t *testing.T) {
	var (
		gotMethod      string
		gotContentType string
		gotPayload     webhookMessage
	)

	client := NewClient("https://discord.example/webhook")
	client.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(bytes.NewReader(nil)),
			Header:     make(http.Header),
		}, nil
	})}
	longSummary := strings.Repeat("あ", maxDescriptionLength+10)
	if err := client.PostSummary("2026-02-23", 42, "一日のタイトル", longSummary); err != nil {
		t.Fatalf("PostSummary() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content-type = %q", gotContentType)
	}
	if gotPayload.Username != "diary-cli" {
		t.Fatalf("Username = %q", gotPayload.Username)
	}
	if len(gotPayload.Embeds) != 1 {
		t.Fatalf("len(Embeds) = %d, want 1", len(gotPayload.Embeds))
	}

	embed := gotPayload.Embeds[0]
	if embed.Title != "2026-02-23 のMisskeyサマリー" {
		t.Fatalf("Title = %q", embed.Title)
	}
	if len([]rune(embed.Description)) != maxDescriptionLength {
		t.Fatalf("Description length = %d, want %d", len([]rune(embed.Description)), maxDescriptionLength)
	}
	if !strings.HasSuffix(embed.Description, "...") {
		t.Fatalf("Description = %q, want truncated suffix", embed.Description)
	}
	if embed.Color != 0x86b300 {
		t.Fatalf("Color = %#x", embed.Color)
	}
	if _, err := time.Parse(time.RFC3339, embed.Timestamp); err != nil {
		t.Fatalf("Timestamp = %q, parse error = %v", embed.Timestamp, err)
	}
	if len(embed.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2", len(embed.Fields))
	}
	if embed.Fields[0] != (discordField{Name: "タイトル", Value: "一日のタイトル"}) {
		t.Fatalf("Fields[0] = %#v", embed.Fields[0])
	}
	if embed.Fields[1] != (discordField{Name: "ノート数", Value: "42", Inline: true}) {
		t.Fatalf("Fields[1] = %#v", embed.Fields[1])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
