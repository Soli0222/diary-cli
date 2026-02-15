package misskey

import (
	"time"
)

const (
	// TIME2000 is January 1, 2000 00:00:00 UTC in milliseconds
	TIME2000 = 946684800000
)

// ParseAID parses a Misskey AID and returns the timestamp
func ParseAID(id string) (time.Time, error) {
	if len(id) < 8 {
		return time.Time{}, &AIDError{Message: "AID too short"}
	}

	// Parse the first 8 characters as base36
	timeStr := id[:8]
	var timeVal int64

	for _, c := range timeStr {
		timeVal *= 36
		if c >= '0' && c <= '9' {
			timeVal += int64(c - '0')
		} else if c >= 'a' && c <= 'z' {
			timeVal += int64(c - 'a' + 10)
		} else {
			return time.Time{}, &AIDError{Message: "invalid character in AID"}
		}
	}

	// Add TIME2000 to get Unix milliseconds
	unixMs := timeVal + TIME2000
	return time.UnixMilli(unixMs), nil
}

// GenAIDPrefix generates the first 8 characters (time portion) of an AID for a given time
func GenAIDPrefix(t time.Time) string {
	ms := t.UnixMilli() - TIME2000
	if ms < 0 {
		ms = 0
	}
	return toBase36(ms, 8)
}

// toBase36 converts a number to base36 string with padding
func toBase36(n int64, pad int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		result := ""
		for i := 0; i < pad; i++ {
			result += "0"
		}
		return result
	}

	result := ""
	for n > 0 {
		result = string(chars[n%36]) + result
		n /= 36
	}

	// Pad with zeros
	for len(result) < pad {
		result = "0" + result
	}

	return result
}

// AIDError represents an error in AID parsing
type AIDError struct {
	Message string
}

func (e *AIDError) Error() string {
	return "AID error: " + e.Message
}
