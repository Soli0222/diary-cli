package preprocess

import (
	"strings"
	"testing"
	"time"

	"github.com/soli0222/diary-cli/internal/models"
)

func strPtr(s string) *string { return &s }

// makeNote creates a note with text at the given UTC time.
func makeNote(id string, text string, utcTime time.Time) models.Note {
	return models.Note{
		ID:        id,
		CreatedAt: utcTime,
		Text:      strPtr(text),
	}
}

// makeRenote creates a pure renote (no text, has renoteID).
func makeRenote(id string, utcTime time.Time) models.Note {
	return models.Note{
		ID:        id,
		CreatedAt: utcTime,
		RenoteID:  strPtr("original"),
	}
}

func TestGroupNotes_TimeGroupingInJST(t *testing.T) {
	// UTC 00:00 = JST 09:00 → 午前
	// UTC 03:00 = JST 12:00 → 午後
	// UTC 08:00 = JST 17:00 → 夕方
	// UTC 12:00 = JST 21:00 → 夜
	// UTC 20:00 = JST 05:00 → 早朝
	notes := []models.Note{
		makeNote("1", "morning note", time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)),  // JST 09:00
		makeNote("2", "afternoon note", time.Date(2026, 2, 15, 3, 0, 0, 0, time.UTC)), // JST 12:00
		makeNote("3", "evening note", time.Date(2026, 2, 15, 8, 0, 0, 0, time.UTC)),   // JST 17:00
		makeNote("4", "night note", time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)),    // JST 21:00
		makeNote("5", "early morning", time.Date(2026, 2, 14, 20, 0, 0, 0, time.UTC)), // JST 05:00
	}

	groups := GroupNotes(notes)

	expectedLabels := []string{
		"早朝 (5:00-9:00)",
		"午前 (9:00-12:00)",
		"午後 (12:00-17:00)",
		"夕方 (17:00-21:00)",
		"夜 (21:00-5:00)",
	}

	if len(groups) != len(expectedLabels) {
		t.Fatalf("GroupNotes() returned %d groups, want %d", len(groups), len(expectedLabels))
	}

	for i, g := range groups {
		if g.Label != expectedLabels[i] {
			t.Errorf("group[%d].Label = %q, want %q", i, g.Label, expectedLabels[i])
		}
		if len(g.Notes) != 1 {
			t.Errorf("group[%d] has %d notes, want 1", i, len(g.Notes))
		}
	}
}

func TestGroupNotes_FiltersPureRenotes(t *testing.T) {
	notes := []models.Note{
		makeNote("1", "original", time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)),
		makeRenote("2", time.Date(2026, 2, 15, 1, 0, 0, 0, time.UTC)),
	}

	groups := GroupNotes(notes)

	totalNotes := 0
	for _, g := range groups {
		totalNotes += len(g.Notes)
	}

	if totalNotes != 1 {
		t.Errorf("GroupNotes() kept %d notes, want 1 (renote should be filtered)", totalNotes)
	}
}

func TestGroupNotes_SortsChronologically(t *testing.T) {
	// Insert in reverse order
	notes := []models.Note{
		makeNote("2", "later", time.Date(2026, 2, 15, 1, 0, 0, 0, time.UTC)),  // JST 10:00
		makeNote("1", "earlier", time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)), // JST 09:00
	}

	groups := GroupNotes(notes)

	// Both should be in 午前 group
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Notes[0].ID != "1" {
		t.Errorf("first note should be ID=1 (earlier), got ID=%s", groups[0].Notes[0].ID)
	}
}

func TestGroupNotes_EmptyInput(t *testing.T) {
	groups := GroupNotes(nil)

	if len(groups) != 0 {
		t.Errorf("GroupNotes(nil) returned %d groups, want 0", len(groups))
	}
}

func TestGroupNotes_EmptyAfterFiltering(t *testing.T) {
	notes := []models.Note{
		makeRenote("1", time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)),
		makeRenote("2", time.Date(2026, 2, 15, 1, 0, 0, 0, time.UTC)),
	}

	groups := GroupNotes(notes)

	if len(groups) != 0 {
		t.Errorf("GroupNotes() with only renotes returned %d groups, want 0", len(groups))
	}
}

func TestGroupNotes_NightGroupBoundary(t *testing.T) {
	// Test boundary: JST 20:59 → 夕方, JST 21:00 → 夜, JST 04:59 → 夜, JST 05:00 → 早朝
	notes := []models.Note{
		makeNote("1", "before night", time.Date(2026, 2, 15, 11, 59, 0, 0, time.UTC)), // JST 20:59 → 夕方
		makeNote("2", "night start", time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)),   // JST 21:00 → 夜
		makeNote("3", "late night", time.Date(2026, 2, 15, 19, 59, 0, 0, time.UTC)),   // JST 04:59 → 夜
		makeNote("4", "early morning", time.Date(2026, 2, 15, 20, 0, 0, 0, time.UTC)), // JST 05:00 → 早朝
	}

	groups := GroupNotes(notes)

	labelMap := map[string]int{}
	for _, g := range groups {
		labelMap[g.Label] = len(g.Notes)
	}

	if labelMap["夕方 (17:00-21:00)"] != 1 {
		t.Errorf("夕方 should have 1 note, got %d", labelMap["夕方 (17:00-21:00)"])
	}
	if labelMap["夜 (21:00-5:00)"] != 2 {
		t.Errorf("夜 should have 2 notes, got %d", labelMap["夜 (21:00-5:00)"])
	}
	if labelMap["早朝 (5:00-9:00)"] != 1 {
		t.Errorf("早朝 should have 1 note, got %d", labelMap["早朝 (5:00-9:00)"])
	}
}

func TestFormatGroupedNotes(t *testing.T) {
	// UTC 00:05 = JST 09:05
	notes := []models.Note{
		makeNote("1", "test note", time.Date(2026, 2, 15, 0, 5, 0, 0, time.UTC)),
	}

	groups := GroupNotes(notes)
	result := FormatGroupedNotes(groups)

	if !strings.Contains(result, "## 午前 (9:00-12:00)") {
		t.Error("expected 午前 header in output")
	}
	if !strings.Contains(result, "- [09:05] test note") {
		t.Errorf("expected JST time 09:05 in output, got:\n%s", result)
	}
}

func TestFormatGroupedNotes_SkipsEmptyText(t *testing.T) {
	notes := []models.Note{
		{
			ID:        "1",
			CreatedAt: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			Text:      strPtr(""),
		},
	}

	groups := GroupNotes(notes)
	result := FormatGroupedNotes(groups)

	// Empty text notes should produce a header but no note lines
	if strings.Contains(result, "- [") {
		t.Error("empty text notes should be skipped in formatting")
	}
}

func TestFormatAllNotes(t *testing.T) {
	// UTC 05:22 = JST 14:22
	notes := []models.Note{
		makeNote("1", "first note", time.Date(2026, 2, 15, 5, 22, 0, 0, time.UTC)),
		makeNote("2", "second note", time.Date(2026, 2, 15, 7, 0, 0, 0, time.UTC)),
		makeRenote("3", time.Date(2026, 2, 15, 8, 0, 0, 0, time.UTC)), // should be filtered
	}

	result := FormatAllNotes(notes)

	if !strings.Contains(result, "- [14:22] first note") {
		t.Errorf("expected JST time 14:22 in output, got:\n%s", result)
	}
	if !strings.Contains(result, "- [16:00] second note") {
		t.Errorf("expected JST time 16:00 in output, got:\n%s", result)
	}
	if strings.Count(result, "- [") != 2 {
		t.Errorf("expected 2 notes (renote filtered), got:\n%s", result)
	}
}

func TestFormatAllNotes_SortsChronologically(t *testing.T) {
	notes := []models.Note{
		makeNote("2", "later", time.Date(2026, 2, 15, 2, 0, 0, 0, time.UTC)),
		makeNote("1", "earlier", time.Date(2026, 2, 15, 1, 0, 0, 0, time.UTC)),
	}

	result := FormatAllNotes(notes)
	lines := strings.Split(strings.TrimSpace(result), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "earlier") {
		t.Errorf("first line should be 'earlier', got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "later") {
		t.Errorf("second line should be 'later', got: %s", lines[1])
	}
}

func TestFormatAllNotes_Empty(t *testing.T) {
	result := FormatAllNotes(nil)
	if result != "" {
		t.Errorf("FormatAllNotes(nil) = %q, want empty string", result)
	}
}

func TestGroupNotes_UTCToJSTConversion(t *testing.T) {
	// This is the exact bug scenario from the user report:
	// UTC 05:05 should display as JST 14:05 (午後)
	notes := []models.Note{
		makeNote("1", "External IP BGP化に成功", time.Date(2026, 2, 15, 5, 5, 22, 0, time.UTC)),
	}

	groups := GroupNotes(notes)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Label != "午後 (12:00-17:00)" {
		t.Errorf("UTC 05:05 (JST 14:05) should be 午後, got %q", groups[0].Label)
	}

	formatted := FormatGroupedNotes(groups)
	if !strings.Contains(formatted, "[14:05]") {
		t.Errorf("expected [14:05] (JST), got:\n%s", formatted)
	}
}
