package profile

import (
	"testing"
	"time"
)

func TestApplyConfirmations_PromoteWhenConfirmedTwice(t *testing.T) {
	t.Parallel()

	base := &UserProfile{
		PendingConfirmations: []PendingConfirmation{{
			Category:      "stable_facts",
			Value:         "朝型",
			Confidence:    0.6,
			Confirmations: 1,
		}},
	}

	updated := ApplyConfirmations(base, []ConfirmationOutcome{{
		Category:  "stable_facts",
		Value:     "朝型",
		Confirmed: true,
	}}, time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC))

	if len(updated.PendingConfirmations) != 0 {
		t.Fatalf("pending len = %d, want 0", len(updated.PendingConfirmations))
	}
	if len(updated.StableFacts) != 1 {
		t.Fatalf("stable_facts len = %d, want 1", len(updated.StableFacts))
	}
	if updated.StableFacts[0].Status != StatusConfirmed {
		t.Fatalf("status = %q, want %q", updated.StableFacts[0].Status, StatusConfirmed)
	}
}

func TestApplyConfirmations_RemoveWhenDenied(t *testing.T) {
	t.Parallel()

	base := &UserProfile{
		PendingConfirmations: []PendingConfirmation{{
			Category:   "ongoing_topics",
			Value:      "転職検討",
			Confidence: 0.7,
		}},
	}

	updated := ApplyConfirmations(base, []ConfirmationOutcome{{
		Category: "ongoing_topics",
		Value:    "転職検討",
		Denied:   true,
	}}, time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC))

	if len(updated.PendingConfirmations) != 0 {
		t.Fatalf("pending len = %d, want 0", len(updated.PendingConfirmations))
	}
	if len(updated.OngoingTopics) != 0 {
		t.Fatalf("ongoing_topics len = %d, want 0", len(updated.OngoingTopics))
	}
	if len(updated.ConfirmationHistory) == 0 {
		t.Fatalf("confirmation history should be recorded")
	}
	if updated.ConfirmationHistory[len(updated.ConfirmationHistory)-1].Result != "denied" {
		t.Fatalf("result = %q, want denied", updated.ConfirmationHistory[len(updated.ConfirmationHistory)-1].Result)
	}
}
