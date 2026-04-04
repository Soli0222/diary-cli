package ai

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIProvider interface {
	Name() string
	Summarize(ctx context.Context, notes string, systemPrompt string) (string, error)
	GenerateTitle(ctx context.Context, summary string) (string, error)
	Chat(ctx context.Context, messages []Message) (string, error)
}
