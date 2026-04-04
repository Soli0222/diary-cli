package ai

import (
	"testing"
	"time"
)

func TestBuildSummaryPrompt(t *testing.T) {
	date := time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC)
	notes := "05時台\n- 朝のノート"

	got := BuildSummaryPrompt(date, notes)
	want := "対象日: 2026-02-23\n\n以下はMisskeyノートを時間帯ごとに整理したものです。1日のサマリーを作成してください。\n\n05時台\n- 朝のノート"

	if got != want {
		t.Fatalf("BuildSummaryPrompt() = %q, want %q", got, want)
	}
}

func TestBuildTitlePrompt(t *testing.T) {
	date := time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC)
	summary := "朝から開発を進めた。"

	got := BuildTitlePrompt(date, summary)
	want := "対象日: 2026-02-23\n\n以下のサマリーにタイトルを付けてください。\n\n朝から開発を進めた。"

	if got != want {
		t.Fatalf("BuildTitlePrompt() = %q, want %q", got, want)
	}
}
