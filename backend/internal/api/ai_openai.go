package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// oaMessage is a Chat Completions message.
type oaMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

type oaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type oaRequest struct {
	Model      string      `json:"model"`
	Messages   []oaMessage `json:"messages"`
	Tools      []oaTool    `json:"tools,omitempty"`
	ToolChoice string      `json:"tool_choice,omitempty"`
}

type oaResponse struct {
	Choices []struct {
		Message oaMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// runOpenAIChat drives the agentic tool-use loop against any OpenAI-compatible
// Chat Completions endpoint (OpenAI, Ollama, Mistral, Groq, LM Studio, vLLM, …).
func (s *Server) runOpenAIChat(ctx context.Context, c *gin.Context, role string, history []aiMessage) (string, []aiStep, error) {
	base := strings.TrimRight(s.cfg.AIBaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	endpoint := base + "/chat/completions"

	tools := make([]oaTool, 0)
	for _, t := range toolsForRole(role) {
		var ot oaTool
		ot.Type = "function"
		ot.Function.Name = t.Name
		ot.Function.Description = t.Description
		ot.Function.Parameters = t.Schema
		tools = append(tools, ot)
	}

	messages := []oaMessage{{Role: "system", Content: aiSystemPrompt}}
	for _, m := range history {
		messages = append(messages, oaMessage{Role: m.Role, Content: m.Content})
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	var steps []aiStep

	for i := 0; i < maxAIIterations; i++ {
		body, _ := json.Marshal(oaRequest{Model: s.cfg.AIModel, Messages: messages, Tools: tools, ToolChoice: "auto"})
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return "", steps, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if s.cfg.AIAPIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+s.cfg.AIAPIKey)
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			return "", steps, err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", steps, fmt.Errorf("AI provider returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		}

		var parsed oaResponse
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return "", steps, err
		}
		if parsed.Error != nil {
			return "", steps, fmt.Errorf("AI provider error: %s", parsed.Error.Message)
		}
		if len(parsed.Choices) == 0 {
			return "", steps, fmt.Errorf("AI provider returned no choices")
		}
		msg := parsed.Choices[0].Message

		if len(msg.ToolCalls) == 0 {
			return msg.Content, steps, nil
		}

		messages = append(messages, msg)
		for _, call := range msg.ToolCalls {
			var args map[string]any
			_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
			out, runErr := s.executeAITool(ctx, c, role, call.Function.Name, args)
			step := aiStep{Tool: call.Function.Name, Input: args, Output: out}
			if runErr != nil {
				step.Error = runErr.Error()
				out = "error: " + runErr.Error()
			}
			steps = append(steps, step)
			messages = append(messages, oaMessage{Role: "tool", ToolCallID: call.ID, Content: out})
		}
	}
	return "Reached the tool-use iteration limit without a final answer.", steps, nil
}
