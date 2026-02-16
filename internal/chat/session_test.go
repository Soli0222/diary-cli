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
	// Verify ratio-based boundaries for maxQ=8 normal: 3:3:2
	if s.phase1End != 3 {
		t.Errorf("phase1End = %d, want 3", s.phase1End)
	}
	if s.phase2End != 6 {
		t.Errorf("phase2End = %d, want 6", s.phase2End)
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
	// Verify ratio-based boundaries for maxQ=8 fewNotes: 2:4:2
	if s.phase1End != 2 {
		t.Errorf("phase1End = %d, want 2", s.phase1End)
	}
	if s.phase2End != 6 {
		t.Errorf("phase2End = %d, want 6", s.phase2End)
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

func TestNewSession_DynamicPromptCounts(t *testing.T) {
	// Normal mode maxQ=8: 3:3:2
	s := NewSession(nil, "notes", 15, 8, 3)
	if !strings.Contains(s.systemPrompt, "事実確認（3問程度）") {
		t.Error("normal maxQ=8 should show 3問程度 for phase1")
	}
	if !strings.Contains(s.systemPrompt, "深掘り（3問程度）") {
		t.Error("normal maxQ=8 should show 3問程度 for phase2")
	}
	if !strings.Contains(s.systemPrompt, "締め（2問程度）") {
		t.Error("normal maxQ=8 should show 2問程度 for phase3")
	}

	// Normal mode maxQ=16: 6:6:4
	s = NewSession(nil, "notes", 15, 16, 3)
	if !strings.Contains(s.systemPrompt, "事実確認（6問程度）") {
		t.Errorf("normal maxQ=16 should show 6問程度 for phase1, prompt: %s", s.systemPrompt)
	}
	if !strings.Contains(s.systemPrompt, "深掘り（6問程度）") {
		t.Errorf("normal maxQ=16 should show 6問程度 for phase2, prompt: %s", s.systemPrompt)
	}
	if !strings.Contains(s.systemPrompt, "締め（4問程度）") {
		t.Errorf("normal maxQ=16 should show 4問程度 for phase3, prompt: %s", s.systemPrompt)
	}

	// Few notes mode maxQ=8: 2:4:2
	s = NewSession(nil, "notes", 5, 8, 3)
	if !strings.Contains(s.systemPrompt, "概要把握（2問程度）") {
		t.Error("fewNotes maxQ=8 should show 2問程度 for phase1")
	}
	if !strings.Contains(s.systemPrompt, "深掘り（4問程度）") {
		t.Error("fewNotes maxQ=8 should show 4問程度 for phase2")
	}
	if !strings.Contains(s.systemPrompt, "締め（2問程度）") {
		t.Error("fewNotes maxQ=8 should show 2問程度 for phase3")
	}
}

func TestGetPhaseHint_NormalMode(t *testing.T) {
	s := &Session{fewNotes: false, maxQuestions: 8, phase1End: 3, phase2End: 6}

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
	s := &Session{fewNotes: true, maxQuestions: 8, phase1End: 2, phase2End: 6}

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
	s := &Session{fewNotes: true, maxQuestions: 8, questionNum: 3, phase1End: 2, phase2End: 6}
	hint := s.getPhaseHint()

	if !strings.Contains(hint, "ノートに表れていない活動") {
		t.Errorf("fewNotes phase2 hint should mention unposted activity, got: %s", hint)
	}
}

func TestGetPhaseHint_FewNotesEarlierTransition(t *testing.T) {
	// fewNotes transitions to phase2 at questionNum=2, normal at questionNum=3
	normalSession := &Session{fewNotes: false, maxQuestions: 8, questionNum: 2, phase1End: 3, phase2End: 6}
	fewNotesSession := &Session{fewNotes: true, maxQuestions: 8, questionNum: 2, phase1End: 2, phase2End: 6}

	normalHint := normalSession.getPhaseHint()
	fewNotesHint := fewNotesSession.getPhaseHint()

	if !strings.Contains(normalHint, "事実確認") {
		t.Error("normal mode at questionNum=2 should still be 事実確認")
	}
	if !strings.Contains(fewNotesHint, "深掘り") {
		t.Error("fewNotes mode at questionNum=2 should already be 深掘り")
	}
}

func TestPhaseBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		maxQ     int
		fewNotes bool
		wantP1   int
		wantP2   int
	}{
		// Normal mode (3:3:2 ratio)
		{"normal_8", 8, false, 3, 6},
		{"normal_12", 12, false, 4, 9},
		{"normal_16", 16, false, 6, 12},
		{"normal_10", 10, false, 3, 7},
		{"normal_3", 3, false, 1, 2},
		{"normal_4", 4, false, 1, 3},
		// Few notes mode (2:4:2 ratio)
		{"fewNotes_8", 8, true, 2, 6},
		{"fewNotes_12", 12, true, 3, 9},
		{"fewNotes_16", 16, true, 4, 12},
		{"fewNotes_10", 10, true, 2, 7},
		{"fewNotes_3", 3, true, 1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p1, p2 := phaseBoundaries(tt.maxQ, tt.fewNotes)
			if p1 != tt.wantP1 {
				t.Errorf("phase1End = %d, want %d", p1, tt.wantP1)
			}
			if p2 != tt.wantP2 {
				t.Errorf("phase2End = %d, want %d", p2, tt.wantP2)
			}

			// Verify all phases get at least 1 question
			phase1Count := p1
			phase2Count := p2 - p1
			phase3Count := tt.maxQ - p2
			if phase1Count < 1 {
				t.Errorf("phase1 count = %d, want >= 1", phase1Count)
			}
			if phase2Count < 1 {
				t.Errorf("phase2 count = %d, want >= 1", phase2Count)
			}
			if phase3Count < 1 {
				t.Errorf("phase3 count = %d, want >= 1", phase3Count)
			}
			// Verify total matches maxQ
			total := phase1Count + phase2Count + phase3Count
			if total != tt.maxQ {
				t.Errorf("total = %d, want %d", total, tt.maxQ)
			}
		})
	}
}

func TestPhaseBoundaries_ClosingPhaseDoesNotGrow(t *testing.T) {
	// The core issue: with hardcoded boundaries, increasing maxQ
	// made the closing phase (phase 3) grow disproportionately.
	// With ratio-based boundaries, phase 3 should stay proportional.

	// Normal mode: 3:3:2 ratio → phase 3 should be ~25% of total
	for _, maxQ := range []int{8, 12, 16, 20, 24} {
		p1, p2 := phaseBoundaries(maxQ, false)
		phase3Count := maxQ - p2
		phase3Ratio := float64(phase3Count) / float64(maxQ)

		// Phase 3 should be roughly 25% (2/8), allow some tolerance for rounding
		if phase3Ratio > 0.35 {
			t.Errorf("normal maxQ=%d: phase3 ratio = %.2f (count=%d), too large (should be ~0.25)",
				maxQ, phase3Ratio, phase3Count)
		}

		_ = p1 // p1 used indirectly via p2
	}

	// Few notes mode: 2:4:2 ratio → phase 3 should also be ~25%
	for _, maxQ := range []int{8, 12, 16, 20, 24} {
		_, p2 := phaseBoundaries(maxQ, true)
		phase3Count := maxQ - p2
		phase3Ratio := float64(phase3Count) / float64(maxQ)

		if phase3Ratio > 0.35 {
			t.Errorf("fewNotes maxQ=%d: phase3 ratio = %.2f (count=%d), too large (should be ~0.25)",
				maxQ, phase3Ratio, phase3Count)
		}
	}
}

func TestGetPhaseHint_ScalesWithMaxQuestions(t *testing.T) {
	// With maxQ=16, normal mode: phase boundaries should be at 6 and 12
	s := &Session{fewNotes: false, maxQuestions: 16, phase1End: 6, phase2End: 12}

	// Question 5 should still be phase 1
	s.questionNum = 5
	hint := s.getPhaseHint()
	if !strings.Contains(hint, "事実確認") {
		t.Errorf("maxQ=16 questionNum=5 should be 事実確認, got: %s", hint)
	}

	// Question 6 should be phase 2
	s.questionNum = 6
	hint = s.getPhaseHint()
	if !strings.Contains(hint, "深掘り") {
		t.Errorf("maxQ=16 questionNum=6 should be 深掘り, got: %s", hint)
	}

	// Question 11 should still be phase 2
	s.questionNum = 11
	hint = s.getPhaseHint()
	if !strings.Contains(hint, "深掘り") {
		t.Errorf("maxQ=16 questionNum=11 should be 深掘り, got: %s", hint)
	}

	// Question 12 should be phase 3
	s.questionNum = 12
	hint = s.getPhaseHint()
	if !strings.Contains(hint, "締め") {
		t.Errorf("maxQ=16 questionNum=12 should be 締め, got: %s", hint)
	}
}
