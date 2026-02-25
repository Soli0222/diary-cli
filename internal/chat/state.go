package chat

import "strings"

// TurnState tracks lightweight per-session understanding signals.
type TurnState struct {
	Unknowns int
}

func (s *TurnState) UpdateFromAnswer(answer string) {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		s.Unknowns++
		return
	}

	a := strings.ToLower(answer)
	uncertainTokens := []string{"わから", "知ら", "しらん", "詳しくは", "直接かかわって", "直接関わって", "覚えてない"}
	for _, token := range uncertainTokens {
		if strings.Contains(a, token) {
			s.Unknowns++
			return
		}
	}

	// Short confirmation answers are often sufficient and should not be treated as "unknown".
	positiveTokens := []string{"はい", "そうです", "その通り", "あってます", "合ってます", "yes", "yep"}
	negativeTokens := []string{"いいえ", "違う", "ちがう", "違います", "not", "no", "そんなことない"}
	for _, token := range positiveTokens {
		if strings.Contains(a, token) {
			if s.Unknowns > 0 {
				s.Unknowns--
			}
			return
		}
	}
	for _, token := range negativeTokens {
		if strings.Contains(a, token) {
			if s.Unknowns > 0 {
				s.Unknowns--
			}
			return
		}
	}

	if len([]rune(answer)) < 8 {
		s.Unknowns++
		return
	}
	if s.Unknowns > 0 {
		s.Unknowns--
	}
}
