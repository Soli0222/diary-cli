package chat

import "testing"

func TestTurnStateUpdateFromAnswer_ShortConfirmationDoesNotIncreaseUnknowns(t *testing.T) {
	t.Parallel()

	var s TurnState
	s.UpdateFromAnswer("そうです")
	if s.Unknowns != 0 {
		t.Fatalf("Unknowns = %d, want 0", s.Unknowns)
	}
}

func TestTurnStateUpdateFromAnswer_UncertainPhraseIncreasesUnknowns(t *testing.T) {
	t.Parallel()

	var s TurnState
	s.UpdateFromAnswer("詳しくはしらない")
	if s.Unknowns != 1 {
		t.Fatalf("Unknowns = %d, want 1", s.Unknowns)
	}
}

func TestTurnStateUpdateFromAnswer_VeryShortNeutralIncreasesUnknowns(t *testing.T) {
	t.Parallel()

	var s TurnState
	s.UpdateFromAnswer("まあ")
	if s.Unknowns != 1 {
		t.Fatalf("Unknowns = %d, want 1", s.Unknowns)
	}
}
