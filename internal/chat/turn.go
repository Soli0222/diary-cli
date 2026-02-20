package chat

import "fmt"

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
	return "今回は要約確認を優先し、『つまり〜という理解で合っていますか？』形式の確認質問をしてください。"
}

func unknownsHint(unknowns int) string {
	return fmt.Sprintf("未確認点が多いため（%d件相当）、新規深掘りよりも確認質問を優先してください。", unknowns)
}

func pendingConfirmationHint(h PendingHypothesis) string {
	return fmt.Sprintf("未確認の仮説があります。次の内容を1つだけ確認してください: [%s] %s。はい/いいえで答えやすい確認質問にしてください。", h.Category, h.Value)
}
