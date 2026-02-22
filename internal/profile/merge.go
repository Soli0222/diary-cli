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
	blocked := mergeConflicts(&merged, updates.Conflicts, dateStr)

	merged.StableFacts = mergeItems("stable_facts", merged.StableFacts, updates.StableFacts, dateStr, &merged.PendingConfirmations, blocked)
	merged.OngoingTopics = mergeItems("ongoing_topics", merged.OngoingTopics, updates.OngoingTopics, dateStr, &merged.PendingConfirmations, blocked)
	merged.EffectivePatterns = mergeItems("effective_patterns", merged.EffectivePatterns, updates.EffectivePatterns, dateStr, &merged.PendingConfirmations, blocked)
	merged.SensitiveTopics = mergeItems("sensitive_topics", merged.SensitiveTopics, updates.SensitiveTopics, dateStr, &merged.PendingConfirmations, blocked)
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

func mergeItems(category string, base, incoming []ProfileItem, dateStr string, pending *[]PendingConfirmation, blocked map[string]struct{}) []ProfileItem {
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
		blockKey := category + ":" + key
		if _, isBlocked := blocked[blockKey]; isBlocked {
			addOrUpdatePending(pending, category, item, dateStr)
			continue
		}
		if item.Status == StatusInferred {
			addOrUpdatePending(pending, category, item, dateStr)
			continue
		}
		if idx, ok := indexByValue[key]; ok {
			existing := result[idx]
			existing.Confidence = math.Max(existing.Confidence, item.Confidence)
			existing.LastSeen = item.LastSeen
			existing.SourceDate = item.SourceDate
			existing.Status = mergeStatus(existing.Status, item.Status)
			result[idx] = existing
			clearPending(pending, category, item.Value)
			continue
		}
		result = append(result, item)
		indexByValue[key] = len(result) - 1
		clearPending(pending, category, item.Value)
	}

	return result
}

func mergeConflicts(profile *UserProfile, incoming []ProfileConflict, dateStr string) map[string]struct{} {
	blocked := map[string]struct{}{}
	if profile == nil || len(incoming) == 0 {
		return blocked
	}

	existingIdx := make(map[string]int, len(profile.Conflicts))
	for i, c := range profile.Conflicts {
		k := conflictKey(c.Category, c.ExistingValue, c.IncomingValue)
		existingIdx[k] = i
	}

	for _, c := range incoming {
		c.Category = strings.TrimSpace(c.Category)
		c.ExistingValue = strings.TrimSpace(c.ExistingValue)
		c.IncomingValue = strings.TrimSpace(c.IncomingValue)
		if c.Category == "" || c.IncomingValue == "" {
			continue
		}
		c.Confidence = clampConfidence(c.Confidence)
		if c.SourceDate == "" {
			c.SourceDate = dateStr
		}
		c.DetectedAt = nowRFC3339()
		k := conflictKey(c.Category, c.ExistingValue, c.IncomingValue)
		if idx, ok := existingIdx[k]; ok {
			prev := profile.Conflicts[idx]
			prev.Confidence = math.Max(prev.Confidence, c.Confidence)
			prev.SourceDate = c.SourceDate
			prev.DetectedAt = c.DetectedAt
			profile.Conflicts[idx] = prev
		} else {
			profile.Conflicts = append(profile.Conflicts, c)
			existingIdx[k] = len(profile.Conflicts) - 1
		}
		blocked[c.Category+":"+normalize(c.IncomingValue)] = struct{}{}
	}
	return blocked
}

func conflictKey(category, existingValue, incomingValue string) string {
	return normalize(category) + "|" + normalize(existingValue) + "|" + normalize(incomingValue)
}

func addOrUpdatePending(pending *[]PendingConfirmation, category string, item ProfileItem, dateStr string) {
	if pending == nil {
		return
	}
	now := nowRFC3339()
	key := normalize(category) + ":" + normalize(item.Value)
	for i := range *pending {
		p := &(*pending)[i]
		if normalize(p.Category)+":"+normalize(p.Value) != key {
			continue
		}
		p.Confidence = math.Max(p.Confidence, item.Confidence)
		p.LastSeen = now
		p.SourceDate = dateStr
		p.Confirmations++
		return
	}

	*pending = append(*pending, PendingConfirmation{
		Category:      category,
		Value:         item.Value,
		Confidence:    item.Confidence,
		FirstSeen:     now,
		LastSeen:      now,
		SourceDate:    dateStr,
		Confirmations: 1,
	})
}

func clearPending(pending *[]PendingConfirmation, category, value string) {
	if pending == nil || len(*pending) == 0 {
		return
	}
	key := normalize(category) + ":" + normalize(value)
	dst := (*pending)[:0]
	for _, p := range *pending {
		if normalize(p.Category)+":"+normalize(p.Value) == key {
			continue
		}
		dst = append(dst, p)
	}
	*pending = dst
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
