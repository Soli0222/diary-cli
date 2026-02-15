package chat

import (
	"strings"
	"testing"
)

func TestNewSession_NormalMode(t *testing.T) {
	s := NewSession(nil, "some notes", 15, 8, 3)

	if s.fewNotes {
		t.Error("noteCount=15 should not be fewNotes")
	}
	if !strings.Contains(s.systemPrompt, "事実確認") {
		t.Error("normal mode should use systemPromptNormal containing 事実確認")
	}
	if strings.Contains(s.systemPrompt, "概要把握") {
		t.Error("normal mode should not contain 概要把握")
	}
	if s.maxQuestions != 8 {
		t.Errorf("maxQuestions = %d, want 8", s.maxQuestions)
	}
	if s.minQuestions != 3 {
		t.Errorf("minQuestions = %d, want 3", s.minQuestions)
	}
}

func TestNewSession_FewNotesMode(t *testing.T) {
	s := NewSession(nil, "few notes", 5, 8, 3)

	if !s.fewNotes {
		t.Error("noteCount=5 should be fewNotes")
	}
	if !strings.Contains(s.systemPrompt, "概要把握") {
		t.Error("fewNotes mode should use systemPromptFewNotes containing 概要把握")
	}
	if !strings.Contains(s.systemPrompt, "重要な方針") {
		t.Error("fewNotes mode should contain 重要な方針 section")
	}
}

func TestNewSession_Threshold(t *testing.T) {
	// Exactly at threshold (10) should be normal
	s := NewSession(nil, "notes", 10, 8, 3)
	if s.fewNotes {
		t.Error("noteCount=10 (at threshold) should not be fewNotes")
	}

	// One below threshold should be fewNotes
	s = NewSession(nil, "notes", 9, 8, 3)
	if !s.fewNotes {
		t.Error("noteCount=9 (below threshold) should be fewNotes")
	}

	// Zero notes
	s = NewSession(nil, "notes", 0, 8, 3)
	if !s.fewNotes {
		t.Error("noteCount=0 should be fewNotes")
	}
}

func TestNewSession_FormattedNotesEmbedded(t *testing.T) {
	notes := "- [14:05] test note content"
	s := NewSession(nil, notes, 5, 8, 3)

	if !strings.Contains(s.systemPrompt, notes) {
		t.Error("systemPrompt should contain the formatted notes")
	}
}

func TestGetPhaseHint_NormalMode(t *testing.T) {
	s := &Session{fewNotes: false}

	tests := []struct {
		questionNum int
		wantContain string
	}{
		{0, "事実確認"},
		{1, "事実確認"},
		{2, "事実確認"},
		{3, "深掘り"},
		{5, "深掘り"},
		{6, "締め"},
		{7, "締め"},
	}

	for _, tt := range tests {
		s.questionNum = tt.questionNum
		hint := s.getPhaseHint()
		if !strings.Contains(hint, tt.wantContain) {
			t.Errorf("normal mode questionNum=%d: hint should contain %q, got %q", tt.questionNum, tt.wantContain, hint)
		}
	}
}

func TestGetPhaseHint_FewNotesMode(t *testing.T) {
	s := &Session{fewNotes: true}

	tests := []struct {
		questionNum int
		wantContain string
	}{
		{0, "概要把握"},
		{1, "概要把握"},
		{2, "深掘り"},
		{3, "深掘り"},
		{5, "深掘り"},
		{6, "締め"},
		{7, "締め"},
	}

	for _, tt := range tests {
		s.questionNum = tt.questionNum
		hint := s.getPhaseHint()
		if !strings.Contains(hint, tt.wantContain) {
			t.Errorf("fewNotes mode questionNum=%d: hint should contain %q, got %q", tt.questionNum, tt.wantContain, hint)
		}
	}
}

func TestGetPhaseHint_FewNotesPhase2MentionsUnpostedActivity(t *testing.T) {
	s := &Session{fewNotes: true, questionNum: 3}
	hint := s.getPhaseHint()

	if !strings.Contains(hint, "ノートに表れていない活動") {
		t.Errorf("fewNotes phase2 hint should mention unposted activity, got: %s", hint)
	}
}

func TestGetPhaseHint_FewNotesEarlierTransition(t *testing.T) {
	// fewNotes transitions to phase2 at questionNum=2, normal at questionNum=3
	normalSession := &Session{fewNotes: false, questionNum: 2}
	fewNotesSession := &Session{fewNotes: true, questionNum: 2}

	normalHint := normalSession.getPhaseHint()
	fewNotesHint := fewNotesSession.getPhaseHint()

	if !strings.Contains(normalHint, "事実確認") {
		t.Error("normal mode at questionNum=2 should still be 事実確認")
	}
	if !strings.Contains(fewNotesHint, "深掘り") {
		t.Error("fewNotes mode at questionNum=2 should already be 深掘り")
	}
}
