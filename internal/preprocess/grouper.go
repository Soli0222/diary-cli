package preprocess

import (
	"fmt"
	"sort"
	"strings"

	"github.com/soli0222/diary-cli/internal/models"
)

// TimeGroup represents a group of notes in a time period.
type TimeGroup struct {
	Label string
	Notes []models.Note
}

// GroupNotes sorts notes chronologically and groups them by time period.
func GroupNotes(notes []models.Note) []TimeGroup {
	// Filter to original notes only (no pure renotes)
	var filtered []models.Note
	for _, n := range notes {
		if n.IsOriginalNote() {
			filtered = append(filtered, n)
		}
	}

	// Sort by CreatedAt ascending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	// Group by time period
	groups := map[string][]models.Note{
		"早朝 (5:00-9:00)":   {},
		"午前 (9:00-12:00)":  {},
		"午後 (12:00-17:00)": {},
		"夕方 (17:00-21:00)": {},
		"夜 (21:00-5:00)":    {},
	}
	order := []string{
		"早朝 (5:00-9:00)",
		"午前 (9:00-12:00)",
		"午後 (12:00-17:00)",
		"夕方 (17:00-21:00)",
		"夜 (21:00-5:00)",
	}

	for _, n := range filtered {
		hour := n.CreatedAt.Hour()
		var label string
		switch {
		case hour >= 5 && hour < 9:
			label = order[0]
		case hour >= 9 && hour < 12:
			label = order[1]
		case hour >= 12 && hour < 17:
			label = order[2]
		case hour >= 17 && hour < 21:
			label = order[3]
		default:
			label = order[4]
		}
		groups[label] = append(groups[label], n)
	}

	var result []TimeGroup
	for _, label := range order {
		if len(groups[label]) > 0 {
			result = append(result, TimeGroup{Label: label, Notes: groups[label]})
		}
	}
	return result
}

// FormatGroupedNotes formats grouped notes into a human-readable string for Claude.
func FormatGroupedNotes(groups []TimeGroup) string {
	var sb strings.Builder
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("## %s\n", g.Label))
		for _, n := range g.Notes {
			ts := n.CreatedAt.Format("15:04")
			text := n.GetDisplayText()
			if text == "" {
				continue
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", ts, text))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatAllNotes formats all notes (flat, chronological) for Claude.
func FormatAllNotes(notes []models.Note) string {
	// Filter and sort
	var filtered []models.Note
	for _, n := range notes {
		if n.IsOriginalNote() {
			filtered = append(filtered, n)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	var sb strings.Builder
	for _, n := range filtered {
		ts := n.CreatedAt.Format("15:04")
		text := n.GetDisplayText()
		if text == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", ts, text))
	}
	return sb.String()
}
