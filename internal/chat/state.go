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
	if len([]rune(answer)) < 20 {
		s.Unknowns++
		return
	}
	if s.Unknowns > 0 {
		s.Unknowns--
	}
}
