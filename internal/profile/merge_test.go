package profile

import (
	"testing"
	"time"
)

func TestMerge_UpsertAndStatusUpgrade(t *testing.T) {
	t.Parallel()

	base := &UserProfile{
		StableFacts: []ProfileItem{{Value: "リモート勤務", Confidence: 0.4, Status: StatusInferred}},
	}
	updates := &CandidateUpdates{
		StableFacts: []ProfileItem{{Value: "リモート勤務", Confidence: 0.8, Status: StatusConfirmed}},
	}

	merged := Merge(base, updates, time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC))
	if len(merged.StableFacts) != 1 {
		t.Fatalf("len = %d, want 1", len(merged.StableFacts))
	}
	if merged.StableFacts[0].Confidence < 0.8 {
		t.Fatalf("confidence = %f, want >= 0.8", merged.StableFacts[0].Confidence)
	}
	if merged.StableFacts[0].Status != StatusConfirmed {
		t.Fatalf("status = %q, want %q", merged.StableFacts[0].Status, StatusConfirmed)
	}
}

func TestMerge_InferredGoesToPending(t *testing.T) {
	t.Parallel()

	base := &UserProfile{}
	updates := &CandidateUpdates{
		StableFacts: []ProfileItem{{Value: "朝型", Confidence: 0.7, Status: StatusInferred}},
	}

	merged := Merge(base, updates, time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC))
	if len(merged.StableFacts) != 0 {
		t.Fatalf("stable_facts len = %d, want 0", len(merged.StableFacts))
	}
	if len(merged.PendingConfirmations) != 1 {
		t.Fatalf("pending_confirmations len = %d, want 1", len(merged.PendingConfirmations))
	}
	if merged.PendingConfirmations[0].Value != "朝型" {
		t.Fatalf("pending value = %q, want %q", merged.PendingConfirmations[0].Value, "朝型")
	}
}

func TestMerge_ConflictBlocksIncoming(t *testing.T) {
	t.Parallel()

	base := &UserProfile{
		StableFacts: []ProfileItem{{Value: "在宅勤務", Confidence: 0.8, Status: StatusObserved}},
	}
	updates := &CandidateUpdates{
		StableFacts: []ProfileItem{{Value: "出社中心", Confidence: 0.8, Status: StatusObserved}},
		Conflicts: []ProfileConflict{
			{
				Category:      "stable_facts",
				ExistingValue: "在宅勤務",
				IncomingValue: "出社中心",
				Confidence:    0.9,
			},
		},
	}

	merged := Merge(base, updates, time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC))
	if len(merged.StableFacts) != 1 {
		t.Fatalf("stable_facts len = %d, want 1", len(merged.StableFacts))
	}
	if len(merged.Conflicts) != 1 {
		t.Fatalf("conflicts len = %d, want 1", len(merged.Conflicts))
	}
	if len(merged.PendingConfirmations) != 1 {
		t.Fatalf("pending_confirmations len = %d, want 1", len(merged.PendingConfirmations))
	}
	if merged.PendingConfirmations[0].Value != "出社中心" {
		t.Fatalf("pending value = %q, want %q", merged.PendingConfirmations[0].Value, "出社中心")
	}
}
