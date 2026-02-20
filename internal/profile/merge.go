package profile

import (
	"math"
	"strings"
	"time"
)

func Merge(base *UserProfile, updates *CandidateUpdates, sourceDate time.Time) *UserProfile {
	if base == nil {
		base = NewEmpty()
	}
	if updates == nil {
		updated := *base
		updated.UpdatedAt = nowRFC3339()
		return &updated
	}

	merged := *base
	dateStr := sourceDate.Format("2006-01-02")

	merged.StableFacts = mergeItems(merged.StableFacts, updates.StableFacts, dateStr)
	merged.OngoingTopics = mergeItems(merged.OngoingTopics, updates.OngoingTopics, dateStr)
	merged.EffectivePatterns = mergeItems(merged.EffectivePatterns, updates.EffectivePatterns, dateStr)
	merged.SensitiveTopics = mergeItems(merged.SensitiveTopics, updates.SensitiveTopics, dateStr)
	merged.Preferences = mergePreferences(merged.Preferences, updates.Preferences)

	applyDecay(merged.StableFacts)
	applyDecay(merged.OngoingTopics)
	applyDecay(merged.EffectivePatterns)
	applyDecay(merged.SensitiveTopics)

	merged.Version = CurrentVersion
	merged.UpdatedAt = nowRFC3339()
	return &merged
}

func mergePreferences(base, incoming UserPreferences) UserPreferences {
	result := base
	if incoming.EmpathyStyle != "" {
		result.EmpathyStyle = incoming.EmpathyStyle
	}
	if incoming.QuestionDepth != "" {
		result.QuestionDepth = incoming.QuestionDepth
	}
	if len(incoming.AvoidTopics) > 0 {
		set := map[string]struct{}{}
		for _, v := range result.AvoidTopics {
			set[v] = struct{}{}
		}
		for _, v := range incoming.AvoidTopics {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			set[v] = struct{}{}
		}
		result.AvoidTopics = result.AvoidTopics[:0]
		for k := range set {
			result.AvoidTopics = append(result.AvoidTopics, k)
		}
	}
	return result
}

func mergeItems(base, incoming []ProfileItem, dateStr string) []ProfileItem {
	if len(incoming) == 0 {
		return base
	}
	result := make([]ProfileItem, len(base))
	copy(result, base)

	indexByValue := make(map[string]int, len(result))
	for i, item := range result {
		indexByValue[normalize(item.Value)] = i
	}

	for _, item := range incoming {
		key := normalize(item.Value)
		if key == "" {
			continue
		}
		item.Value = strings.TrimSpace(item.Value)
		item.Confidence = clampConfidence(item.Confidence)
		if item.Status == "" {
			item.Status = StatusObserved
		}
		if item.SourceDate == "" {
			item.SourceDate = dateStr
		}
		item.LastSeen = nowRFC3339()
		if idx, ok := indexByValue[key]; ok {
			existing := result[idx]
			existing.Confidence = math.Max(existing.Confidence, item.Confidence)
			existing.LastSeen = item.LastSeen
			existing.SourceDate = item.SourceDate
			existing.Status = mergeStatus(existing.Status, item.Status)
			result[idx] = existing
			continue
		}
		result = append(result, item)
		indexByValue[key] = len(result) - 1
	}

	return result
}

func applyDecay(items []ProfileItem) {
	now := time.Now()
	for i := range items {
		if items[i].LastSeen == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, items[i].LastSeen)
		if err != nil {
			continue
		}
		if now.Sub(t) > 30*24*time.Hour {
			items[i].Confidence = clampConfidence(items[i].Confidence * 0.95)
		}
	}
}

func mergeStatus(oldStatus, newStatus string) string {
	if newStatus == StatusConfirmed || oldStatus == StatusConfirmed {
		return StatusConfirmed
	}
	if newStatus == StatusObserved || oldStatus == StatusObserved {
		return StatusObserved
	}
	return StatusInferred
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func clampConfidence(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	if v == 0 {
		return 0.5
	}
	return v
}
