package cli

import "testing"

func TestSafeRate(t *testing.T) {
	t.Parallel()
	if got := safeRate(1, 4); got != 0.25 {
		t.Fatalf("safeRate(1,4) = %v, want 0.25", got)
	}
	if got := safeRate(1, 0); got != 0 {
		t.Fatalf("safeRate(1,0) = %v, want 0", got)
	}
}
