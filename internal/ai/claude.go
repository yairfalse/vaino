package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type ClaudeClient struct {
	client anthropic.Client
}

func NewClaudeClient() (*ClaudeClient, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &ClaudeClient{client: client}, nil
}

func (c *ClaudeClient) AnalyzeDrift(ctx context.Context, driftData string) (string, error) {
	prompt := fmt.Sprintf(`You are an expert infrastructure engineer. Analyze the following infrastructure drift data and provide insights:

%s

Please provide:
1. A summary of the key changes detected
2. Risk assessment (High/Medium/Low) for each change
3. Recommended actions to address the drift
4. Potential impact on system reliability and security

Format your response in a clear, actionable manner.`, driftData)

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5Sonnet20241022,
		MaxTokens: anthropic.Int(1024),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to analyze drift: %w", err)
	}

	if len(resp.Content) > 0 {
		if textBlock := resp.Content[0].Text; textBlock != "" {
			return textBlock, nil
		}
	}

	return "No response content received", nil
}

func (c *ClaudeClient) ExplainChange(ctx context.Context, changeData string) (string, error) {
	prompt := fmt.Sprintf(`You are an infrastructure expert. Explain the following infrastructure change in simple terms:

%s

Please explain:
1. What exactly changed
2. Why this change might have occurred
3. What the implications are for the system
4. Whether this is a normal operational change or something that needs attention

Use clear, non-technical language where possible.`, changeData)

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5Sonnet20241022,
		MaxTokens: anthropic.Int(512),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to explain change: %w", err)
	}

	if len(resp.Content) > 0 {
		if textBlock := resp.Content[0].Text; textBlock != "" {
			return textBlock, nil
		}
	}

	return "No response content received", nil
}

func (c *ClaudeClient) GenerateRemediation(ctx context.Context, driftData string) (string, error) {
	prompt := fmt.Sprintf(`You are an infrastructure automation expert. Based on the following drift data, provide specific remediation steps:

%s

Please provide:
1. Step-by-step remediation instructions
2. Commands or scripts that can be used
3. Verification steps to ensure the fix worked
4. Prevention measures to avoid similar drift in the future

Focus on practical, actionable solutions.`, driftData)

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5Sonnet20241022,
		MaxTokens: anthropic.Int(1024),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate remediation: %w", err)
	}

	if len(resp.Content) > 0 {
		if textBlock := resp.Content[0].Text; textBlock != "" {
			return textBlock, nil
		}
	}

	return "No response content received", nil
}