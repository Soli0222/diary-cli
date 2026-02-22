package profile

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "profile.json")
	in := &UserProfile{
		Version:     CurrentVersion,
		StableFacts: []ProfileItem{{Value: "朝型", Confidence: 0.9, Status: StatusConfirmed}},
	}
	if err := Save(path, in); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(out.StableFacts) != 1 {
		t.Fatalf("stable_facts len = %d, want 1", len(out.StableFacts))
	}
	if out.StableFacts[0].Value != "朝型" {
		t.Fatalf("stable_facts[0].Value = %q", out.StableFacts[0].Value)
	}
}
