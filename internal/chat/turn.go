package chat

import (
	"encoding/json"
	"fmt"
	"strings"
)

type turnResponse struct {
	Intent             string `json:"intent,omitempty"`
	SummaryCheck       bool   `json:"summary_check,omitempty"`
	EmpathyLine        string `json:"empathy_line,omitempty"`
	Question           string `json:"question"`
	ReasoningNote      string `json:"reasoning_note,omitempty"`
	StateUpdateHint    string `json:"state_update_hint,omitempty"`
	ConfirmationTarget struct {
		Category string `json:"category,omitempty"`
		Value    string `json:"value,omitempty"`
	} `json:"confirmation_target,omitempty"`
}

type confirmationJudgeResponse struct {
	Result string `json:"result"`
	Reason string `json:"reason"`
}

func empathyHint(style string) string {
	switch style {
	case "deep":
		return "感情の言語化を優先し、受容的で丁寧な問い方をしてください。"
	case "light":
		return "簡潔で軽いトーンを保ちつつ、相手の発話を尊重してください。"
	default:
		return "ユーザーの回答を受け止める一言を含意した質問にしてください。"
	}
}

func summaryCheckHint() string {
	return "今回は要約確認を優先してください。言い換え確認は短く自然に行い、同じ要約表現の反復は避けてください。"
}

func unknownsHint(unknowns int) string {
	return fmt.Sprintf("未確認点が多いため（%d件相当）、新規深掘りよりも確認質問を優先してください。", unknowns)
}

func pendingConfirmationHint(h PendingHypothesis) string {
	return fmt.Sprintf("未確認の仮説があります。次の内容を1つだけ確認してください: [%s] %s。はい/いいえで答えやすい確認質問にしてください。", h.Category, h.Value)
}

func questionQualityHint() string {
	return "ユーザーが言っていない前提（環境・役割・機材・属性など）を勝手に追加しないでください。推測が必要なら断定せず確認として聞いてください。答えにくい抽象質問（例: 位置づけ/意味づけだけを問う質問）を避け、具体的に何を答えればよいか分かる質問にしてください。ユーザーが『詳しく知らない』『直接関わっていない』と答えた話題は、その前提で受け止めて別の切り口へ移ってください。"
}

func turnSchemaInstruction() string {
	return `必ずJSONのみを返してください。Markdownや説明文は禁止です。
出力スキーマ:
{
  "intent": "fact_check|deep_dive|summary|confirm_hypothesis",
  "summary_check": true/false,
  "empathy_line": "（任意）内部メモ。質問本文には含めない",
  "question": "ユーザーに見せる質問文（1つだけ）",
  "reasoning_note": "（任意）短い内部メモ",
  "state_update_hint": "（任意）短い内部メモ",
  "confirmation_target": {"category":"...","value":"..."}
}
制約:
- question には1つの質問だけを書く
- 日本語で書く
- 前置きや解説は不要
- ユーザー未提示の事実を断定前提にしない
- 答えにくい抽象質問より、具体的な質問を優先する`
}

func parseTurnResponse(raw string) (turnResponse, error) {
	var out turnResponse
	clean := cleanupJSON(raw)
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		return turnResponse{}, err
	}
	out.Question = normalizeQuestion(out.Question)
	if out.Question == "" {
		return turnResponse{}, fmt.Errorf("empty question")
	}
	return out, nil
}

func fallbackQuestion(raw string) string {
	clean := strings.TrimSpace(cleanupJSON(raw))
	return normalizeQuestion(clean)
}

func cleanupJSON(raw string) string {
	t := strings.TrimSpace(raw)
	t = strings.TrimPrefix(t, "```json")
	t = strings.TrimPrefix(t, "```")
	t = strings.TrimSuffix(t, "```")
	return strings.TrimSpace(t)
}

func normalizeQuestion(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	lines := strings.Split(q, "\n")
	q = strings.TrimSpace(lines[0])
	if idx := strings.Index(q, "？"); idx >= 0 {
		return strings.TrimSpace(q[:idx+len("？")])
	}
	if idx := strings.Index(q, "?"); idx >= 0 {
		return strings.TrimSpace(q[:idx+len("?")])
	}
	return q
}

func parseConfirmationJudgeResponse(raw string) (confirmationVerdict, error) {
	var out confirmationJudgeResponse
	if err := json.Unmarshal([]byte(cleanupJSON(raw)), &out); err != nil {
		return confirmationVerdict{}, err
	}

	switch strings.ToLower(strings.TrimSpace(out.Result)) {
	case "confirmed":
		return confirmationVerdict{Confirmed: true, Method: "llm", Reason: strings.TrimSpace(out.Reason)}, nil
	case "denied":
		return confirmationVerdict{Denied: true, Method: "llm", Reason: strings.TrimSpace(out.Reason)}, nil
	default:
		return confirmationVerdict{Uncertain: true, Method: "llm", Reason: strings.TrimSpace(out.Reason)}, nil
	}
}
