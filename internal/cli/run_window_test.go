package cli

import (
	"testing"
	"time"
)

func TestResolveDiaryWindow(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("JST", 9*60*60)
	targetDate := time.Date(2026, 2, 22, 0, 0, 0, 0, loc)

	start, end := resolveDiaryWindow(targetDate)

	wantStart := time.Date(2026, 2, 22, 5, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 2, 23, 5, 0, 0, 0, loc)

	if !start.Equal(wantStart) {
		t.Fatalf("start = %v, want %v", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Fatalf("end = %v, want %v", end, wantEnd)
	}
}
