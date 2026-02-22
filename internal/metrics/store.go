package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type RunMetrics struct {
	Date                  string  `json:"date"`
	RecordedAt            string  `json:"recorded_at"`
	QuestionsTotal        int     `json:"questions_total"`
	SummaryCheckTurns     int     `json:"summary_check_turns"`
	StructuredTurns       int     `json:"structured_turns"`
	FallbackTurns         int     `json:"fallback_turns"`
	ConfirmationAttempts  int     `json:"confirmation_attempts"`
	ConfirmationConfirmed int     `json:"confirmation_confirmed"`
	ConfirmationDenied    int     `json:"confirmation_denied"`
	ConfirmationUncertain int     `json:"confirmation_uncertain"`
	AvgAnswerLength       float64 `json:"avg_answer_length"`
	DuplicateQuestionRate float64 `json:"duplicate_question_rate"`
	StableFactsBefore     int     `json:"stable_facts_before"`
	StableFactsAfter      int     `json:"stable_facts_after"`
	PendingBefore         int     `json:"pending_before"`
	PendingAfter          int     `json:"pending_after"`
	ConflictsBefore       int     `json:"conflicts_before"`
	ConflictsAfter        int     `json:"conflicts_after"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "diary-cli", "metrics.jsonl"), nil
}

func Append(path string, item RunMetrics) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	if item.RecordedAt == "" {
		item.RecordedAt = time.Now().Format(time.RFC3339)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create metrics dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer func() { _ = f.Close() }()

	b, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("failed to append metrics: %w", err)
	}
	return nil
}

func LoadSince(path string, since time.Time) ([]RunMetrics, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var out []RunMetrics
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Bytes()
		if len(line) == 0 {
			continue
		}
		var item RunMetrics
		if err := json.Unmarshal(line, &item); err != nil {
			continue
		}
		if item.Date != "" {
			d, err := time.ParseInLocation("2006-01-02", item.Date, since.Location())
			if err == nil && d.Before(since) {
				continue
			}
		}
		out = append(out, item)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}
	return out, nil
}
