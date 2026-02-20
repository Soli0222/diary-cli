package profile

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/soli0222/diary-cli/internal/claude"
)

const learningSystemPrompt = `あなたは対話ログからユーザープロファイル更新候補を抽出するアシスタントです。
必ずJSONのみを返してください。Markdownや説明文は不要です。

出力JSONスキーマ:
{
  "stable_facts": [{"value":"...","confidence":0.0,"status":"observed|inferred|confirmed"}],
  "preferences": {"empathy_style":"light|balanced|deep","question_depth":"light|balanced|deep","avoid_topics":["..."]},
  "ongoing_topics": [{"value":"...","confidence":0.0,"status":"observed|inferred|confirmed"}],
  "effective_patterns": [{"value":"...","confidence":0.0,"status":"observed|inferred|confirmed"}],
  "sensitive_topics": [{"value":"...","confidence":0.0,"status":"observed|inferred|confirmed"}],
  "conflicts": [{"category":"stable_facts|ongoing_topics|effective_patterns|sensitive_topics","existing_value":"...","incoming_value":"...","confidence":0.0}]
}

抽出ルール:
- 推測よりも会話中に明示された内容を優先する
- 新規性の低い項目は出力しない
- 個人情報の過剰な具体化は避ける
- confidenceは0.0-1.0
- 既存プロファイルと矛盾する候補は該当カテゴリの配列には含めず、conflictsにのみ出力する
`

func ExtractUpdates(client *claude.Client, conversation []claude.Message, date time.Time, current *UserProfile) (*CandidateUpdates, error) {
	if client == nil {
		return nil, fmt.Errorf("client is nil")
	}
	if len(conversation) == 0 {
		return &CandidateUpdates{}, nil
	}

	var sb strings.Builder
	sb.WriteString("以下は当日の対話ログです。新規に学習すべき要素のみ抽出してください。\n")
	sb.WriteString("日付: ")
	sb.WriteString(date.Format("2006-01-02"))
	sb.WriteString("\n\n")
	sb.WriteString("既存プロファイル（JSON）:\n")
	sb.WriteString(profileForLearningPrompt(current))
	sb.WriteString("\n\n")
	for _, msg := range conversation {
		role := "ユーザー"
		if msg.Role == "assistant" {
			role = "インタビュアー"
		}
		sb.WriteString(role)
		sb.WriteString(": ")
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}

	resp, err := client.Chat(learningSystemPrompt, []claude.Message{{Role: "user", Content: sb.String()}})
	if err != nil {
		return nil, err
	}

	jsonText := cleanupJSON(resp)
	var updates CandidateUpdates
	if err := json.Unmarshal([]byte(jsonText), &updates); err != nil {
		return nil, fmt.Errorf("failed to parse learning response: %w", err)
	}

	normalizeUpdates(&updates, date)
	return &updates, nil
}

func cleanupJSON(raw string) string {
	t := strings.TrimSpace(raw)
	t = strings.TrimPrefix(t, "```json")
	t = strings.TrimPrefix(t, "```")
	t = strings.TrimSuffix(t, "```")
	return strings.TrimSpace(t)
}

func normalizeUpdates(u *CandidateUpdates, date time.Time) {
	if u == nil {
		return
	}
	dateStr := date.Format("2006-01-02")
	norm := func(items []ProfileItem) {
		for i := range items {
			items[i].Value = strings.TrimSpace(items[i].Value)
			items[i].Confidence = clampConfidence(items[i].Confidence)
			if items[i].Status == "" {
				items[i].Status = StatusObserved
			}
			if items[i].SourceDate == "" {
				items[i].SourceDate = dateStr
			}
		}
	}
	norm(u.StableFacts)
	norm(u.OngoingTopics)
	norm(u.EffectivePatterns)
	norm(u.SensitiveTopics)
	for i := range u.Conflicts {
		u.Conflicts[i].Category = strings.TrimSpace(u.Conflicts[i].Category)
		u.Conflicts[i].ExistingValue = strings.TrimSpace(u.Conflicts[i].ExistingValue)
		u.Conflicts[i].IncomingValue = strings.TrimSpace(u.Conflicts[i].IncomingValue)
		u.Conflicts[i].Confidence = clampConfidence(u.Conflicts[i].Confidence)
		if u.Conflicts[i].SourceDate == "" {
			u.Conflicts[i].SourceDate = dateStr
		}
	}

	if u.Preferences.EmpathyStyle != "" && u.Preferences.EmpathyStyle != "light" && u.Preferences.EmpathyStyle != "balanced" && u.Preferences.EmpathyStyle != "deep" {
		u.Preferences.EmpathyStyle = ""
	}
	if u.Preferences.QuestionDepth != "" && u.Preferences.QuestionDepth != "light" && u.Preferences.QuestionDepth != "balanced" && u.Preferences.QuestionDepth != "deep" {
		u.Preferences.QuestionDepth = ""
	}
}

func profileForLearningPrompt(current *UserProfile) string {
	if current == nil {
		return "{}"
	}
	data, err := json.Marshal(current)
	if err != nil {
		return "{}"
	}
	return string(data)
}
