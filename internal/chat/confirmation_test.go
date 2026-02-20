package chat

import "testing"

func TestClassifyConfirmationAnswer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		answer    string
		confirmed bool
		denied    bool
	}{
		{name: "positive_ja", answer: "はい、その通りです", confirmed: true, denied: false},
		{name: "negative_ja", answer: "いいえ、違います", confirmed: false, denied: true},
		{name: "neutral", answer: "ちょっとわからない", confirmed: false, denied: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confirmed, denied := classifyConfirmationAnswer(tt.answer)
			if confirmed != tt.confirmed || denied != tt.denied {
				t.Fatalf("classify(%q) = (%v,%v), want (%v,%v)", tt.answer, confirmed, denied, tt.confirmed, tt.denied)
			}
		})
	}
}
