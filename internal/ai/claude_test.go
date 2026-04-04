package ai

import (
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

func TestAnthropicMessages(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "system prompt"},
		{Role: "developer", Content: "developer prompt"},
		{Role: "user", Content: "user prompt"},
		{Role: "assistant", Content: "assistant reply"},
		{Role: "other", Content: "fallback user"},
		{Role: "user", Content: "   "},
	}

	system, got := anthropicMessages(messages)

	if len(system) != 1 {
		t.Fatalf("len(system) = %d, want 1", len(system))
	}
	if system[0].Text != "system prompt\n\ndeveloper prompt" {
		t.Fatalf("system[0].Text = %q", system[0].Text)
	}

	if len(got) != 3 {
		t.Fatalf("len(messages) = %d, want 3", len(got))
	}

	assertAnthropicMessage(t, got[0], anthropic.MessageParamRoleUser, "user prompt")
	assertAnthropicMessage(t, got[1], anthropic.MessageParamRoleAssistant, "assistant reply")
	assertAnthropicMessage(t, got[2], anthropic.MessageParamRoleUser, "fallback user")
}

func assertAnthropicMessage(t *testing.T, got anthropic.MessageParam, wantRole anthropic.MessageParamRole, wantText string) {
	t.Helper()

	if got.Role != wantRole {
		t.Fatalf("role = %q, want %q", got.Role, wantRole)
	}
	if len(got.Content) != 1 {
		t.Fatalf("len(content) = %d, want 1", len(got.Content))
	}
	if got.Content[0].OfText == nil {
		t.Fatalf("content[0] is not a text block: %#v", got.Content[0])
	}
	if got.Content[0].OfText.Text != wantText {
		t.Fatalf("text = %q, want %q", got.Content[0].OfText.Text, wantText)
	}
}
