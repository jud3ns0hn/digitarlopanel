package api

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/gin-gonic/gin"
)

// runAnthropicChat drives the agentic tool-use loop against the Claude Messages
// API using the official Anthropic Go SDK.
func (s *Server) runAnthropicChat(ctx context.Context, c *gin.Context, role string, history []aiMessage) (string, []aiStep, error) {
	opts := []option.RequestOption{option.WithAPIKey(s.cfg.AIAPIKey)}
	if s.cfg.AIBaseURL != "" {
		opts = append(opts, option.WithBaseURL(s.cfg.AIBaseURL))
	}
	client := anthropic.NewClient(opts...)

	// Build the tool definitions for this role.
	tools := make([]anthropic.ToolUnionParam, 0)
	for _, t := range toolsForRole(role) {
		tools = append(tools, anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: t.Schema["properties"],
			},
		}})
	}

	messages := make([]anthropic.MessageParam, 0, len(history))
	for _, m := range history {
		if m.Role == "assistant" {
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		} else {
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		}
	}

	var steps []aiStep
	for i := 0; i < maxAIIterations; i++ {
		resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(s.cfg.AIModel),
			MaxTokens: 4096,
			System:    []anthropic.TextBlockParam{{Text: aiSystemPrompt}},
			Tools:     tools,
			Messages:  messages,
		})
		if err != nil {
			return "", steps, err
		}
		messages = append(messages, resp.ToParam())

		if resp.StopReason != anthropic.StopReasonToolUse {
			return collectAnthropicText(resp), steps, nil
		}

		toolResults := make([]anthropic.ContentBlockParamUnion, 0)
		for _, block := range resp.Content {
			tu, ok := block.AsAny().(anthropic.ToolUseBlock)
			if !ok {
				continue
			}
			var args map[string]any
			_ = json.Unmarshal([]byte(tu.JSON.Input.Raw()), &args)
			out, runErr := s.executeAITool(ctx, c, role, tu.Name, args)
			step := aiStep{Tool: tu.Name, Input: args, Output: out}
			isErr := false
			if runErr != nil {
				step.Error = runErr.Error()
				out = runErr.Error()
				isErr = true
			}
			steps = append(steps, step)
			toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, out, isErr))
		}
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}
	return "Reached the tool-use iteration limit without a final answer.", steps, nil
}

func collectAnthropicText(resp *anthropic.Message) string {
	out := ""
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			out += t.Text
		}
	}
	return out
}
