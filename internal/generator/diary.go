package generator

import (
	"fmt"
	"strings"
	"time"

	"github.com/soli0222/diary-cli/internal/models"
)

type JSONOutput struct {
	Date      string           `json:"date"`
	StartTime string           `json:"start_time"`
	EndTime   string           `json:"end_time"`
	NoteCount int              `json:"note_count"`
	Title     string           `json:"title"`
	Summary   string           `json:"summary"`
	Notes     []JSONOutputNote `json:"notes"`
}

type JSONOutputNote struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	Text      string `json:"text"`
}

func BuildMarkdown(date time.Time, author, title, summary string) string {
	dateStr := date.Format("2006-01-02")
	timeStr := date.Format("2006-01-02T15:04")

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "title: %s\n", dateStr)
	fmt.Fprintf(&sb, "author: %s\n", author)
	sb.WriteString("layout: post\n")
	fmt.Fprintf(&sb, "date: %s\n", timeStr)
	sb.WriteString("category: 日記\n")
	sb.WriteString("---\n\n")
	fmt.Fprintf(&sb, "# %s\n\n", title)
	sb.WriteString("# Misskeyサマリー\n\n")
	sb.WriteString(strings.TrimSpace(summary))
	sb.WriteString("\n")

	return sb.String()
}

func BuildSummaryText(date time.Time, noteCount int, title, summary string) string {
	return fmt.Sprintf(
		"%s のサマリー\nノート数: %d\nタイトル: %s\n\n%s",
		date.Format("2006-01-02"),
		noteCount,
		title,
		strings.TrimSpace(summary),
	)
}

func BuildJSONOutput(targetDate, startTime, endTime time.Time, title, summary string, notes []models.Note) JSONOutput {
	items := make([]JSONOutputNote, 0, len(notes))
	for _, note := range notes {
		items = append(items, JSONOutputNote{
			ID:        note.ID,
			CreatedAt: note.CreatedAt.Format(time.RFC3339),
			Text:      note.GetDisplayText(),
		})
	}

	return JSONOutput{
		Date:      targetDate.Format("2006-01-02"),
		StartTime: startTime.Format(time.RFC3339),
		EndTime:   endTime.Format(time.RFC3339),
		NoteCount: len(notes),
		Title:     title,
		Summary:   summary,
		Notes:     items,
	}
}
