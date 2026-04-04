package ai

import (
	"context"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
)

const claudeDefaultModel = "claude-sonnet-4-6"

type ClaudeProvider struct {
	client anthropic.Client
	model  string
}

func NewClaudeProvider(apiKey, model string) *ClaudeProvider {
	if model == "" {
		model = claudeDefaultModel
	}
	return &ClaudeProvider{
		client: anthropic.NewClient(anthropicoption.WithAPIKey(apiKey)),
		model:  model,
	}
}

func (p *ClaudeProvider) Name() string {
	return "claude"
}

func (p *ClaudeProvider) Summarize(ctx context.Context, notes string, systemPrompt string) (string, error) {
	return p.Chat(ctx, []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: notes},
	})
}

func (p *ClaudeProvider) GenerateTitle(ctx context.Context, summary string) (string, error) {
	return p.Chat(ctx, []Message{
		{Role: "system", Content: titleSystemPrompt},
		{Role: "user", Content: summary},
	})
}

func (p *ClaudeProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	systemBlocks, chatMessages := anthropicMessages(messages)

	response, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096,
		System:    systemBlocks,
		Messages:  chatMessages,
	})
	if err != nil {
		return "", fmt.Errorf("anthropic messages API failed: %w", err)
	}

	var parts []string
	for _, block := range response.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, block.Text)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("anthropic returned no text content")
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}

func anthropicMessages(messages []Message) ([]anthropic.TextBlockParam, []anthropic.MessageParam) {
	var (
		systemParts []string
		out         []anthropic.MessageParam
	)

	for _, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		switch role {
		case "system", "developer":
			systemParts = append(systemParts, content)
		case "assistant":
			out = append(out, anthropic.NewAssistantMessage(anthropic.NewTextBlock(content)))
		default:
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(content)))
		}
	}

	var system []anthropic.TextBlockParam
	if len(systemParts) > 0 {
		system = []anthropic.TextBlockParam{{Text: strings.Join(systemParts, "\n\n")}}
	}
	return system, out
}
