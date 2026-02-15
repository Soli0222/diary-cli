package generator

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMarkdown(t *testing.T) {
	date := time.Date(2026, 2, 15, 21, 30, 0, 0, time.FixedZone("Asia/Tokyo", 9*60*60))
	author := "TestUser"
	title := "テストの一日"
	diaryBody := "今日はテストを書いた。"
	summary := "テストに関するサマリー。"

	result := BuildMarkdown(date, author, title, diaryBody, summary)

	// Check frontmatter
	checks := []struct {
		label    string
		expected string
	}{
		{"frontmatter start", "---\n"},
		{"title field", "title: 2026-02-15\n"},
		{"author field", "author: TestUser\n"},
		{"layout field", "layout: post\n"},
		{"date field", "date: 2026-02-15T21:30\n"},
		{"category field", "category: 日記\n"},
		{"heading", "# テストの一日\n"},
		{"diary body", "今日はテストを書いた。"},
		{"summary heading", "# Misskeyサマリー\n"},
		{"summary body", "テストに関するサマリー。"},
	}

	for _, c := range checks {
		if !strings.Contains(result, c.expected) {
			t.Errorf("BuildMarkdown() missing %s: %q\nGot:\n%s", c.label, c.expected, result)
		}
	}
}

func TestBuildMarkdown_EmptyDiaryBody(t *testing.T) {
	date := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	result := BuildMarkdown(date, "User", "Title", "", "Summary")

	// Should still contain the structure
	if !strings.Contains(result, "# Title\n") {
		t.Error("should contain title heading")
	}
	if !strings.Contains(result, "# Misskeyサマリー\n") {
		t.Error("should contain summary heading")
	}
}

func TestBuildMarkdown_FrontmatterStructure(t *testing.T) {
	date := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	result := BuildMarkdown(date, "Author", "Title", "Body", "Summary")

	// Verify frontmatter is properly delimited
	parts := strings.SplitN(result, "---", 3)
	if len(parts) < 3 {
		t.Fatal("frontmatter should be delimited by ---")
	}

	frontmatter := parts[1]
	if !strings.Contains(frontmatter, "title:") {
		t.Error("frontmatter should contain title")
	}
	if !strings.Contains(frontmatter, "author:") {
		t.Error("frontmatter should contain author")
	}
	if !strings.Contains(frontmatter, "date:") {
		t.Error("frontmatter should contain date")
	}
}

func TestBuildMarkdown_DateFormatting(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		wantDate string
		wantTime string
	}{
		{
			name:     "single digit month and day",
			date:     time.Date(2026, 1, 5, 8, 5, 0, 0, time.UTC),
			wantDate: "title: 2026-01-05",
			wantTime: "date: 2026-01-05T08:05",
		},
		{
			name:     "double digit month and day",
			date:     time.Date(2026, 12, 25, 23, 59, 0, 0, time.UTC),
			wantDate: "title: 2026-12-25",
			wantTime: "date: 2026-12-25T23:59",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildMarkdown(tt.date, "Author", "Title", "Body", "Summary")
			if !strings.Contains(result, tt.wantDate) {
				t.Errorf("expected %q in output", tt.wantDate)
			}
			if !strings.Contains(result, tt.wantTime) {
				t.Errorf("expected %q in output", tt.wantTime)
			}
		})
	}
}

func TestBuildMarkdown_TrailingNewline(t *testing.T) {
	result := BuildMarkdown(time.Now(), "Author", "Title", "Body", "Summary")

	if !strings.HasSuffix(result, "\n") {
		t.Error("output should end with newline")
	}
}
