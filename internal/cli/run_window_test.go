package cli

import (
	"testing"
	"time"

	"github.com/soli0222/diary-cli/internal/profile"
)

func TestResolveCollectionWindowForRun(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("JST", 9*60*60)
	targetDate := time.Date(2026, 2, 22, 0, 0, 0, 0, loc)
	now := time.Date(2026, 2, 23, 2, 0, 0, 0, loc)

	t.Run("explicit date ignores profile updated_at", func(t *testing.T) {
		prof := &profile.UserProfile{UpdatedAt: "2026-02-23T01:00:00+09:00"}
		start, end, ok, reason := resolveCollectionWindowForRun(now, targetDate, true, prof)
		if ok {
			t.Fatal("ok = true, want false")
		}
		if reason != "explicit_date" {
			t.Fatalf("reason = %q, want explicit_date", reason)
		}
		wantStart := time.Date(2026, 2, 22, 0, 0, 0, 0, loc)
		wantEnd := time.Date(2026, 2, 23, 0, 0, 0, 0, loc)
		if !start.Equal(wantStart) || !end.Equal(wantEnd) {
			t.Fatalf("window = [%v, %v), want [%v, %v)", start, end, wantStart, wantEnd)
		}
	})

	t.Run("nil profile falls back", func(t *testing.T) {
		_, _, ok, reason := resolveCollectionWindowForRun(now, targetDate, false, nil)
		if ok {
			t.Fatal("ok = true, want false")
		}
		if reason != "nil_profile" {
			t.Fatalf("reason = %q, want nil_profile", reason)
		}
	})

	t.Run("empty updated_at falls back", func(t *testing.T) {
		prof := &profile.UserProfile{}
		_, _, ok, reason := resolveCollectionWindowForRun(now, targetDate, false, prof)
		if ok {
			t.Fatal("ok = true, want false")
		}
		if reason != "empty_updated_at" {
			t.Fatalf("reason = %q, want empty_updated_at", reason)
		}
	})

	t.Run("uses profile updated_at when valid", func(t *testing.T) {
		prof := &profile.UserProfile{UpdatedAt: "2026-02-22T22:30:00+09:00"}
		start, end, ok, reason := resolveCollectionWindowForRun(now, targetDate, false, prof)
		if !ok {
			t.Fatalf("ok = false, reason = %q", reason)
		}
		wantStart := time.Date(2026, 2, 22, 22, 30, 0, 0, loc)
		if !start.Equal(wantStart) {
			t.Fatalf("start = %v, want %v", start, wantStart)
		}
		if !end.Equal(now) {
			t.Fatalf("end = %v, want %v", end, now)
		}
	})

	t.Run("invalid updated_at falls back", func(t *testing.T) {
		prof := &profile.UserProfile{UpdatedAt: "not-a-time"}
		_, _, ok, reason := resolveCollectionWindowForRun(now, targetDate, false, prof)
		if ok {
			t.Fatal("ok = true, want false")
		}
		if reason != "invalid_updated_at" {
			t.Fatalf("reason = %q, want invalid_updated_at", reason)
		}
	})

	t.Run("future or equal updated_at falls back", func(t *testing.T) {
		prof := &profile.UserProfile{UpdatedAt: now.Format(time.RFC3339)}
		_, _, ok, reason := resolveCollectionWindowForRun(now, targetDate, false, prof)
		if ok {
			t.Fatal("ok = true, want false")
		}
		if reason != "future_or_equal_updated_at" {
			t.Fatalf("reason = %q, want future_or_equal_updated_at", reason)
		}
	})
}
