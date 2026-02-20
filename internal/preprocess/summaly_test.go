package preprocess

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soli0222/diary-cli/internal/models"
)

func noteTextPtr(s string) *string { return &s }

func TestExtractSummalyTargets(t *testing.T) {
	text := "ref https://example.com/a, dup https://example.com/a and spotify https://open.spotify.com/track/123 and other https://go.dev/doc"
	got := ExtractSummalyTargets(text)

	want := []string{"https://example.com/a", "https://go.dev/doc"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d (%v)", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsSpotifyURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{url: "https://open.spotify.com/track/abc", want: true},
		{url: "https://www.spotify.com/jp/", want: true},
		{url: "https://spotify.com/", want: true},
		{url: "https://example.com/spotify.com", want: false},
	}

	for _, tt := range tests {
		if got := IsSpotifyURL(tt.url); got != tt.want {
			t.Fatalf("IsSpotifyURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestEnrichNotesWithSummaly(t *testing.T) {
	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("url")
		calls[raw]++

		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprintf(w, `{"title":"Page for %s","description":"desc for %s","sitename":"Example","url":"%s"}`,
			raw, raw, raw); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := NewSummalyClientWithEndpoint(server.URL)

	notes := []models.Note{
		{
			ID:        "1",
			CreatedAt: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
			Text:      noteTextPtr("look https://example.com/a and https://open.spotify.com/track/123"),
		},
		{
			ID:        "2",
			CreatedAt: time.Date(2026, 2, 20, 1, 0, 0, 0, time.UTC),
			Text:      noteTextPtr("same link again https://example.com/a"),
		},
	}

	got := EnrichNotesWithSummaly(notes, client)

	if !strings.Contains(*got[0].Text, "[link summary]") {
		t.Fatalf("first note should contain link summary, got: %q", *got[0].Text)
	}
	if !strings.Contains(*got[0].Text, "Page for https://example.com/a") {
		t.Fatalf("expected summary title in first note, got: %q", *got[0].Text)
	}
	if strings.Contains(*got[0].Text, "Page for https://open.spotify.com/track/123") {
		t.Fatalf("spotify URL should not be summarized, got: %q", *got[0].Text)
	}
	if !strings.Contains(*got[1].Text, "Page for https://example.com/a") {
		t.Fatalf("expected summary title in second note, got: %q", *got[1].Text)
	}

	if calls["https://example.com/a"] != 1 {
		t.Fatalf("summaly should be called once per unique URL, got %d", calls["https://example.com/a"])
	}
	if calls["https://open.spotify.com/track/123"] != 0 {
		t.Fatalf("spotify URL should not be called, got %d", calls["https://open.spotify.com/track/123"])
	}
}

func TestEnrichNotesWithSummaly_SkipWhenClientNil(t *testing.T) {
	original := "check https://example.com/a"
	notes := []models.Note{
		{
			ID:        "1",
			CreatedAt: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
			Text:      noteTextPtr(original),
		},
	}

	got := EnrichNotesWithSummaly(notes, nil)
	if got[0].Text == nil || *got[0].Text != original {
		t.Fatalf("note text should remain unchanged when client is nil, got: %v", got[0].Text)
	}
}
