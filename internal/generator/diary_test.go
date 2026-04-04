package generator

import (
	"strings"
	"testing"
	"time"

	"github.com/soli0222/diary-cli/internal/models"
)

func TestBuildMarkdown(t *testing.T) {
	date := time.Date(2026, 2, 15, 5, 0, 0, 0, time.FixedZone("Asia/Tokyo", 9*60*60))
	result := BuildMarkdown(date, "TestUser", "テストの一日", "テストに関するサマリー。")

	checks := []string{
		"---\n",
		"title: 2026-02-15\n",
		"author: TestUser\n",
		"layout: post\n",
		"date: 2026-02-15T05:00\n",
		"category: 日記\n",
		"# テストの一日\n",
		"# Misskeyサマリー\n",
		"テストに関するサマリー。",
	}

	for _, expected := range checks {
		if !strings.Contains(result, expected) {
			t.Fatalf("BuildMarkdown() missing %q\nGot:\n%s", expected, result)
		}
	}
}

func TestBuildMarkdown_DoesNotIncludeDiaryBodySection(t *testing.T) {
	date := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	result := BuildMarkdown(date, "User", "Title", "Summary")

	if strings.Count(result, "# ") != 2 {
		t.Fatalf("expected exactly two headings, got:\n%s", result)
	}
}

func TestBuildSummaryText(t *testing.T) {
	date := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	got := BuildSummaryText(date, 42, "タイトル", "本文")

	for _, expected := range []string{"2026-02-15 のサマリー", "ノート数: 42", "タイトル: タイトル", "本文"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q in %q", expected, got)
		}
	}
}

func TestBuildJSONOutput(t *testing.T) {
	start := time.Date(2026, 2, 15, 5, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	text := "note"
	note := models.Note{
		ID:        "abc",
		CreatedAt: start,
		Text:      &text,
	}

	got := BuildJSONOutput(start, start, end, "Title", "Summary", []models.Note{note})

	if got.Date != "2026-02-15" {
		t.Fatalf("Date = %q", got.Date)
	}
	if got.NoteCount != 1 {
		t.Fatalf("NoteCount = %d", got.NoteCount)
	}
	if len(got.Notes) != 1 || got.Notes[0].ID != "abc" || got.Notes[0].Text != "note" {
		t.Fatalf("unexpected notes: %#v", got.Notes)
	}
}
