package cli

import (
	"testing"
	"time"
)

func TestResolveTargetDate(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("JST", 9*60*60)
	now := time.Date(2026, 2, 23, 2, 15, 30, 0, loc)

	t.Run("default uses today at midnight", func(t *testing.T) {
		got, err := resolveTargetDate(now, "", false)
		if err != nil {
			t.Fatalf("resolveTargetDate() error = %v", err)
		}
		want := time.Date(2026, 2, 23, 0, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("got = %v, want %v", got, want)
		}
	})

	t.Run("yesterday shifts date only", func(t *testing.T) {
		got, err := resolveTargetDate(now, "", true)
		if err != nil {
			t.Fatalf("resolveTargetDate() error = %v", err)
		}
		want := time.Date(2026, 2, 22, 0, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("got = %v, want %v", got, want)
		}
	})

	t.Run("date flag wins over yesterday", func(t *testing.T) {
		got, err := resolveTargetDate(now, "2026-02-15", true)
		if err != nil {
			t.Fatalf("resolveTargetDate() error = %v", err)
		}
		want := time.Date(2026, 2, 15, 0, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("got = %v, want %v", got, want)
		}
	})

	t.Run("invalid date format", func(t *testing.T) {
		if _, err := resolveTargetDate(now, "2026/02/15", false); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
