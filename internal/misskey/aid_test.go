package misskey

import (
	"testing"
	"time"
)

func TestParseAID(t *testing.T) {
	tests := []struct {
		name    string
		aid     string
		wantErr bool
	}{
		{
			name:    "valid AID",
			aid:     "9k0zw73b0h",
			wantErr: false,
		},
		{
			name:    "too short",
			aid:     "abc",
			wantErr: true,
		},
		{
			name:    "invalid character",
			aid:     "ABCD1234xy",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAID(tt.aid)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenAIDPrefix(t *testing.T) {
	// Test that the generated prefix is 8 characters
	now := time.Now()
	prefix := GenAIDPrefix(now)

	if len(prefix) != 8 {
		t.Errorf("GenAIDPrefix() length = %d, want 8", len(prefix))
	}

	// Verify that the prefix can be parsed back
	parsed, err := ParseAID(prefix + "00") // Add dummy noise
	if err != nil {
		t.Errorf("ParseAID() failed to parse generated prefix: %v", err)
	}

	// The parsed time should be within a few seconds of the original
	diff := now.Sub(parsed)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("Parsed time differs by %v, want less than 1 second", diff)
	}
}

func TestAIDRoundTrip(t *testing.T) {
	// Test a round trip: generate prefix -> parse -> compare
	testTime := time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC)
	prefix := GenAIDPrefix(testTime)
	parsed, err := ParseAID(prefix + "00")
	if err != nil {
		t.Fatalf("ParseAID() error = %v", err)
	}

	// Times should be within a millisecond (since AID uses milliseconds)
	diff := testTime.Sub(parsed)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Millisecond {
		t.Errorf("Round trip time difference = %v, want < 1ms", diff)
	}
}

func TestToBase36(t *testing.T) {
	tests := []struct {
		n        int64
		pad      int
		expected string
	}{
		{0, 8, "00000000"},
		{35, 2, "0z"},
		{36, 2, "10"},
		{1000, 4, "00rs"},
	}

	for _, tt := range tests {
		result := toBase36(tt.n, tt.pad)
		if result != tt.expected {
			t.Errorf("toBase36(%d, %d) = %s, want %s", tt.n, tt.pad, result, tt.expected)
		}
	}
}
