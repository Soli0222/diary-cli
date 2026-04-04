package cli

import (
	"testing"
	"time"
)

func TestResolveTargetDate(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("JST", 9*60*60)

	t.Run("default after 5am uses current diary date", func(t *testing.T) {
		now := time.Date(2026, 2, 23, 8, 15, 30, 0, loc)
		got, err := resolveTargetDate(now, "", false, loc)
		if err != nil {
			t.Fatalf("resolveTargetDate() error = %v", err)
		}
		want := time.Date(2026, 2, 23, 0, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("got = %v, want %v", got, want)
		}
	})

	t.Run("default before 5am uses previous diary date", func(t *testing.T) {
		now := time.Date(2026, 2, 23, 2, 15, 30, 0, loc)
		got, err := resolveTargetDate(now, "", false, loc)
		if err != nil {
			t.Fatalf("resolveTargetDate() error = %v", err)
		}
		want := time.Date(2026, 2, 22, 0, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("got = %v, want %v", got, want)
		}
	})

	t.Run("yesterday shifts from diary date", func(t *testing.T) {
		now := time.Date(2026, 2, 23, 2, 15, 30, 0, loc)
		got, err := resolveTargetDate(now, "", true, loc)
		if err != nil {
			t.Fatalf("resolveTargetDate() error = %v", err)
		}
		want := time.Date(2026, 2, 21, 0, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("got = %v, want %v", got, want)
		}
	})

	t.Run("date flag wins over yesterday", func(t *testing.T) {
		now := time.Date(2026, 2, 23, 2, 15, 30, 0, loc)
		got, err := resolveTargetDate(now, "2026-02-15", true, loc)
		if err != nil {
			t.Fatalf("resolveTargetDate() error = %v", err)
		}
		want := time.Date(2026, 2, 15, 0, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("got = %v, want %v", got, want)
		}
	})

	t.Run("invalid date format", func(t *testing.T) {
		if _, err := resolveTargetDate(time.Date(2026, 2, 23, 8, 15, 30, 0, loc), "2026/02/15", false, loc); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
