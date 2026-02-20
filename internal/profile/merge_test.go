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
