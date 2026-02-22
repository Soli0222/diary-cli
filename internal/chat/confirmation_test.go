package chat

import "testing"

func TestClassifyConfirmationAnswer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		answer    string
		confirmed bool
		denied    bool
		uncertain bool
	}{
		{name: "positive_ja", answer: "はい、その通りです", confirmed: true, denied: false, uncertain: false},
		{name: "negative_ja", answer: "いいえ、違います", confirmed: false, denied: true, uncertain: false},
		{name: "neutral", answer: "ちょっとわからない", confirmed: false, denied: false, uncertain: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := classifyConfirmationAnswer(tt.answer)
			if v.Confirmed != tt.confirmed || v.Denied != tt.denied || v.Uncertain != tt.uncertain {
				t.Fatalf("classify(%q) = (%v,%v,%v), want (%v,%v,%v)", tt.answer, v.Confirmed, v.Denied, v.Uncertain, tt.confirmed, tt.denied, tt.uncertain)
			}
			if v.Method == "" {
				t.Fatalf("method should not be empty")
			}
		})
	}
}
