package ai

import (
	"testing"

	"google.golang.org/genai"
)

func TestGeminiRequest(t *testing.T) {
	contents, config := geminiRequest([]Message{
		{Role: "system", Content: "system prompt"},
		{Role: "developer", Content: "developer prompt"},
		{Role: "user", Content: "user prompt"},
		{Role: "assistant", Content: "assistant reply"},
		{Role: "other", Content: "fallback user"},
		{Role: "user", Content: "   "},
	})

	if config.SystemInstruction == nil {
		t.Fatal("SystemInstruction is nil")
	}
	if len(config.SystemInstruction.Parts) != 1 {
		t.Fatalf("len(SystemInstruction.Parts) = %d, want 1", len(config.SystemInstruction.Parts))
	}
	if config.SystemInstruction.Parts[0].Text != "system prompt\n\ndeveloper prompt" {
		t.Fatalf("SystemInstruction text = %q", config.SystemInstruction.Parts[0].Text)
	}

	if len(contents) != 3 {
		t.Fatalf("len(contents) = %d, want 3", len(contents))
	}

	assertGeminiContent(t, contents[0], genai.RoleUser, "user prompt")
	assertGeminiContent(t, contents[1], genai.RoleModel, "assistant reply")
	assertGeminiContent(t, contents[2], genai.RoleUser, "fallback user")
}

func assertGeminiContent(t *testing.T, got *genai.Content, wantRole genai.Role, wantText string) {
	t.Helper()

	if got == nil {
		t.Fatal("content is nil")
	}
	if got.Role != string(wantRole) {
		t.Fatalf("role = %q, want %q", got.Role, wantRole)
	}
	if len(got.Parts) != 1 {
		t.Fatalf("len(parts) = %d, want 1", len(got.Parts))
	}
	if got.Parts[0].Text != wantText {
		t.Fatalf("text = %q, want %q", got.Parts[0].Text, wantText)
	}
}
