package profile

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const CurrentVersion = 1

const (
	StatusObserved  = "observed"
	StatusInferred  = "inferred"
	StatusConfirmed = "confirmed"
)

type ProfileItem struct {
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
	LastSeen   string  `json:"last_seen,omitempty"`
	SourceDate string  `json:"source_date,omitempty"`
	Status     string  `json:"status,omitempty"`
}

type UserPreferences struct {
	EmpathyStyle  string   `json:"empathy_style,omitempty"`
	QuestionDepth string   `json:"question_depth,omitempty"`
	AvoidTopics   []string `json:"avoid_topics,omitempty"`
}

type UserProfile struct {
	Version              int                   `json:"version"`
	UpdatedAt            string                `json:"updated_at,omitempty"`
	StableFacts          []ProfileItem         `json:"stable_facts,omitempty"`
	Preferences          UserPreferences       `json:"preferences,omitempty"`
	OngoingTopics        []ProfileItem         `json:"ongoing_topics,omitempty"`
	EffectivePatterns    []ProfileItem         `json:"effective_patterns,omitempty"`
	SensitiveTopics      []ProfileItem         `json:"sensitive_topics,omitempty"`
	Conflicts            []ProfileConflict     `json:"conflicts,omitempty"`
	PendingConfirmations []PendingConfirmation `json:"pending_confirmations,omitempty"`
	ConfirmationHistory  []ConfirmationRecord  `json:"confirmation_history,omitempty"`
}

type ProfileConflict struct {
	Category      string  `json:"category"`
	ExistingValue string  `json:"existing_value"`
	IncomingValue string  `json:"incoming_value"`
	Confidence    float64 `json:"confidence,omitempty"`
	SourceDate    string  `json:"source_date,omitempty"`
	DetectedAt    string  `json:"detected_at,omitempty"`
	Resolved      bool    `json:"resolved,omitempty"`
}

type PendingConfirmation struct {
	Category      string  `json:"category"`
	Value         string  `json:"value"`
	Confidence    float64 `json:"confidence"`
	FirstSeen     string  `json:"first_seen,omitempty"`
	LastSeen      string  `json:"last_seen,omitempty"`
	SourceDate    string  `json:"source_date,omitempty"`
	Confirmations int     `json:"confirmations,omitempty"`
}

type CandidateUpdates struct {
	StableFacts       []ProfileItem     `json:"stable_facts,omitempty"`
	Preferences       UserPreferences   `json:"preferences,omitempty"`
	OngoingTopics     []ProfileItem     `json:"ongoing_topics,omitempty"`
	EffectivePatterns []ProfileItem     `json:"effective_patterns,omitempty"`
	SensitiveTopics   []ProfileItem     `json:"sensitive_topics,omitempty"`
	Conflicts         []ProfileConflict `json:"conflicts,omitempty"`
}

func NewEmpty() *UserProfile {
	return &UserProfile{Version: CurrentVersion}
}

func SummaryForPrompt(p *UserProfile, maxItems int) string {
	if p == nil {
		return ""
	}
	if maxItems <= 0 {
		maxItems = 8
	}

	var lines []string
	appendItems := func(title string, items []ProfileItem) {
		if len(items) == 0 {
			return
		}
		sorted := make([]ProfileItem, len(items))
		copy(sorted, items)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Confidence > sorted[j].Confidence
		})
		limit := min(len(sorted), maxItems)
		var vals []string
		for i := 0; i < limit; i++ {
			vals = append(vals, fmt.Sprintf("%s(%.2f)", sorted[i].Value, sorted[i].Confidence))
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", title, strings.Join(vals, ", ")))
	}

	appendItems("継続している事実", p.StableFacts)
	appendItems("継続トピック", p.OngoingTopics)
	appendItems("有効だった質問傾向", p.EffectivePatterns)
	appendItems("慎重に扱う話題", p.SensitiveTopics)
	if len(p.PendingConfirmations) > 0 {
		limit := min(len(p.PendingConfirmations), maxItems)
		vals := make([]string, 0, limit)
		for i := 0; i < limit; i++ {
			item := p.PendingConfirmations[i]
			vals = append(vals, fmt.Sprintf("%s:%s(%.2f)", item.Category, item.Value, item.Confidence))
		}
		lines = append(lines, fmt.Sprintf("- 未確認の仮説（確認優先）: %s", strings.Join(vals, ", ")))
	}

	if p.Preferences.EmpathyStyle != "" || p.Preferences.QuestionDepth != "" || len(p.Preferences.AvoidTopics) > 0 {
		pref := []string{}
		if p.Preferences.EmpathyStyle != "" {
			pref = append(pref, "共感スタイル="+p.Preferences.EmpathyStyle)
		}
		if p.Preferences.QuestionDepth != "" {
			pref = append(pref, "質問深度="+p.Preferences.QuestionDepth)
		}
		if len(p.Preferences.AvoidTopics) > 0 {
			pref = append(pref, "避ける話題="+strings.Join(p.Preferences.AvoidTopics, ","))
		}
		lines = append(lines, "- 嗜好: "+strings.Join(pref, "; "))
	}

	if len(lines) == 0 {
		return ""
	}

	return "以下はこれまでのユーザープロファイルです。既知情報の重複質問は避け、変化点を優先してください。\n" + strings.Join(lines, "\n")
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}
