package metrics

import (
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndLoadSince(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "metrics.jsonl")
	if err := Append(path, RunMetrics{Date: "2026-02-20", QuestionsTotal: 5}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := Append(path, RunMetrics{Date: "2026-02-10", QuestionsTotal: 3}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	since := time.Date(2026, 2, 15, 0, 0, 0, 0, time.Local)
	items, err := LoadSince(path, since)
	if err != nil {
		t.Fatalf("LoadSince() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].Date != "2026-02-20" {
		t.Fatalf("date = %q", items[0].Date)
	}
}
