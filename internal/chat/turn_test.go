package chat

import "testing"

func TestParseTurnResponse_JSON(t *testing.T) {
	t.Parallel()

	raw := `{"intent":"deep_dive","summary_check":false,"question":"それはなぜだと思いますか？"}`
	res, err := parseTurnResponse(raw)
	if err != nil {
		t.Fatalf("parseTurnResponse error = %v", err)
	}
	if res.Question != "それはなぜだと思いますか？" {
		t.Fatalf("question = %q", res.Question)
	}
}

func TestFallbackQuestion_Text(t *testing.T) {
	t.Parallel()

	raw := "まず背景をありがとうございます。次に、そのときどう感じましたか？ もし可能なら理由も教えてください。"
	q := fallbackQuestion(raw)
	if q == "" {
		t.Fatalf("fallback question should not be empty")
	}
}
