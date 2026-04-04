package ai

import "testing"

func TestOpenAIMessages(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "system prompt"},
		{Role: "developer", Content: "developer prompt"},
		{Role: "user", Content: "user prompt"},
		{Role: "assistant", Content: "assistant reply"},
		{Role: "other", Content: "fallback user"},
		{Role: "user", Content: "   "},
	}

	got := openAIMessages(messages)

	if len(got) != 5 {
		t.Fatalf("len(messages) = %d, want 5", len(got))
	}

	if got[0].OfDeveloper == nil || got[0].OfDeveloper.Content.OfString.Value != "system prompt" {
		t.Fatalf("message[0] = %#v", got[0])
	}
	if got[1].OfDeveloper == nil || got[1].OfDeveloper.Content.OfString.Value != "developer prompt" {
		t.Fatalf("message[1] = %#v", got[1])
	}
	if got[2].OfUser == nil || got[2].OfUser.Content.OfString.Value != "user prompt" {
		t.Fatalf("message[2] = %#v", got[2])
	}
	if got[3].OfAssistant == nil || got[3].OfAssistant.Content.OfString.Value != "assistant reply" {
		t.Fatalf("message[3] = %#v", got[3])
	}
	if got[4].OfUser == nil || got[4].OfUser.Content.OfString.Value != "fallback user" {
		t.Fatalf("message[4] = %#v", got[4])
	}
}
