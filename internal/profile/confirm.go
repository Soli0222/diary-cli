package profile

import (
	"math"
	"strings"
	"time"
)

type ConfirmationOutcome struct {
	QuestionNum int
	Category    string
	Value       string
	Question    string
	Answer      string
	Confirmed   bool
	Denied      bool
}

type ConfirmationRecord struct {
	Category   string `json:"category"`
	Value      string `json:"value"`
	Result     string `json:"result"`
	Question   string `json:"question,omitempty"`
	Answer     string `json:"answer,omitempty"`
	RecordedAt string `json:"recorded_at,omitempty"`
}

func ApplyConfirmations(base *UserProfile, outcomes []ConfirmationOutcome, date time.Time) *UserProfile {
	if base == nil {
		base = NewEmpty()
	}
	if len(outcomes) == 0 {
		return base
	}

	updated := *base
	dateStr := date.Format("2006-01-02")

	for _, out := range outcomes {
		category := strings.TrimSpace(out.Category)
		value := strings.TrimSpace(out.Value)
		if category == "" || value == "" {
			continue
		}

		if out.Denied {
			updated.PendingConfirmations = removePending(updated.PendingConfirmations, category, value)
			updated.addConfirmationRecord(ConfirmationRecord{
				Category:   category,
				Value:      value,
				Result:     "denied",
				Question:   out.Question,
				Answer:     out.Answer,
				RecordedAt: nowRFC3339(),
			})
			continue
		}

		if !out.Confirmed {
			continue
		}

		pending, ok := findPending(updated.PendingConfirmations, category, value)
		if !ok {
			continue
		}
		pending.Confirmations++
		pending.LastSeen = nowRFC3339()
		pending.SourceDate = dateStr

		if pending.Confirmations < 2 && pending.Confidence < 0.75 {
			updated.PendingConfirmations = upsertPending(updated.PendingConfirmations, pending)
			updated.addConfirmationRecord(ConfirmationRecord{
				Category:   category,
				Value:      value,
				Result:     "partial_confirmed",
				Question:   out.Question,
				Answer:     out.Answer,
				RecordedAt: nowRFC3339(),
			})
			continue
		}

		item := ProfileItem{
			Value:      value,
			Confidence: math.Max(0.8, clampConfidence(pending.Confidence)),
			LastSeen:   nowRFC3339(),
			SourceDate: dateStr,
			Status:     StatusConfirmed,
		}
		updated.promoteItem(category, item)
		updated.PendingConfirmations = removePending(updated.PendingConfirmations, category, value)
		updated.addConfirmationRecord(ConfirmationRecord{
			Category:   category,
			Value:      value,
			Result:     "confirmed",
			Question:   out.Question,
			Answer:     out.Answer,
			RecordedAt: nowRFC3339(),
		})
	}

	updated.UpdatedAt = nowRFC3339()
	return &updated
}

func (p *UserProfile) promoteItem(category string, item ProfileItem) {
	switch category {
	case "stable_facts":
		p.StableFacts = mergeItems(category, p.StableFacts, []ProfileItem{item}, item.SourceDate, &p.PendingConfirmations, nil)
	case "ongoing_topics":
		p.OngoingTopics = mergeItems(category, p.OngoingTopics, []ProfileItem{item}, item.SourceDate, &p.PendingConfirmations, nil)
	case "effective_patterns":
		p.EffectivePatterns = mergeItems(category, p.EffectivePatterns, []ProfileItem{item}, item.SourceDate, &p.PendingConfirmations, nil)
	case "sensitive_topics":
		p.SensitiveTopics = mergeItems(category, p.SensitiveTopics, []ProfileItem{item}, item.SourceDate, &p.PendingConfirmations, nil)
	}
}

func removePending(items []PendingConfirmation, category, value string) []PendingConfirmation {
	if len(items) == 0 {
		return items
	}
	key := normalize(category) + ":" + normalize(value)
	dst := items[:0]
	for _, item := range items {
		if normalize(item.Category)+":"+normalize(item.Value) == key {
			continue
		}
		dst = append(dst, item)
	}
	return dst
}

func findPending(items []PendingConfirmation, category, value string) (PendingConfirmation, bool) {
	key := normalize(category) + ":" + normalize(value)
	for _, item := range items {
		if normalize(item.Category)+":"+normalize(item.Value) == key {
			return item, true
		}
	}
	return PendingConfirmation{}, false
}

func upsertPending(items []PendingConfirmation, target PendingConfirmation) []PendingConfirmation {
	key := normalize(target.Category) + ":" + normalize(target.Value)
	for i := range items {
		if normalize(items[i].Category)+":"+normalize(items[i].Value) == key {
			items[i] = target
			return items
		}
	}
	return append(items, target)
}

func (p *UserProfile) addConfirmationRecord(rec ConfirmationRecord) {
	p.ConfirmationHistory = append(p.ConfirmationHistory, rec)
	const maxHistory = 200
	if len(p.ConfirmationHistory) > maxHistory {
		p.ConfirmationHistory = p.ConfirmationHistory[len(p.ConfirmationHistory)-maxHistory:]
	}
}
