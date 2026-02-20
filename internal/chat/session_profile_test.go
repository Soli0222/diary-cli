package chat

import (
	"strings"
	"testing"
)

func TestNewSessionWithOptions_IncludesProfileSummary(t *testing.T) {
	t.Parallel()

	s := NewSessionWithOptions(nil, "notes", 12, 8, 3, Options{ProfileSummary: "- 継続トピック: 健康"})
	if !strings.Contains(s.systemPrompt, "ユーザープロファイル") {
		t.Fatalf("system prompt should include profile section")
	}
	if !strings.Contains(s.systemPrompt, "継続トピック") {
		t.Fatalf("system prompt should include profile summary text")
	}
}

func TestSession_ShouldSummaryCheck(t *testing.T) {
	t.Parallel()

	s := &Session{summaryEvery: 2, questionNum: 2}
	if !s.shouldSummaryCheck() {
		t.Fatalf("shouldSummaryCheck() = false, want true")
	}
}

func TestSession_ShouldConfirmUnknowns(t *testing.T) {
	t.Parallel()

	s := &Session{maxUnknownsBeforeConfirm: 3, state: TurnState{Unknowns: 3}}
	if !s.shouldConfirmUnknowns() {
		t.Fatalf("shouldConfirmUnknowns() = false, want true")
	}
}
