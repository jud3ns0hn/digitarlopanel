package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

// maxAIIterations caps the agentic tool-use loop.
const maxAIIterations = 8

const aiSystemPrompt = `You are the assistant built into DigitarloPanel, a Linux server administration panel.
Help the operator inspect and safely manage their server using the provided tools.
Prefer reading state (metrics, logs, lists) before acting. Use control actions only when asked.
You cannot delete data, run arbitrary shell commands, edit files, or manage panel users — those are intentionally not available to you.
Be concise and practical. When you report results, lead with the answer.`

type aiChatRequest struct {
	Messages []aiMessage `json:"messages" binding:"required"`
}

type aiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// aiStep records one tool execution for the response trace.
type aiStep struct {
	Tool   string `json:"tool"`
	Input  any    `json:"input"`
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

func (s *Server) aiConfigured() bool {
	return s.cfg.AIAPIKey != "" || s.cfg.AIProvider == "openai"
}

func (s *Server) handleAIStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"configured": s.aiConfigured(),
		"provider":   s.cfg.AIProvider,
		"model":      s.cfg.AIModel,
	})
}

func (s *Server) handleAIChat(c *gin.Context) {
	if !s.aiConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI assistant is not configured (set provider, model and API key in settings)"})
		return
	}
	var req aiChatRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Messages) == 0 {
		badRequest(c, "messages required")
		return
	}

	_, _, role := currentUser(c)
	s.audit(c, "ai_chat", req.Messages[len(req.Messages)-1].Content)

	ctx := c.Request.Context()
	var (
		reply string
		steps []aiStep
		err   error
	)
	switch s.cfg.AIProvider {
	case "openai":
		reply, steps, err = s.runOpenAIChat(ctx, c, role, req.Messages)
	default:
		reply, steps, err = s.runAnthropicChat(ctx, c, role, req.Messages)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "steps": steps})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reply": reply, "steps": steps})
}

// toolsForRole returns the tools available to a given role: write tools are
// hidden from viewers so the model cannot attempt state changes it can't make.
func toolsForRole(role string) []aiTool {
	all := aiTools()
	if role == model.RoleAdmin || role == model.RoleOperator {
		return all
	}
	out := make([]aiTool, 0, len(all))
	for _, t := range all {
		if !t.Write {
			out = append(out, t)
		}
	}
	return out
}

// executeAITool runs a tool by name with RBAC enforcement and auditing. It is
// the single choke point both provider loops call, so security is uniform.
func (s *Server) executeAITool(ctx context.Context, c *gin.Context, role, name string, args map[string]any) (string, error) {
	tool, ok := findAITool(name)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	if tool.Write && role != model.RoleAdmin && role != model.RoleOperator {
		return "", fmt.Errorf("permission denied: %s requires operator or admin role", name)
	}
	out, err := tool.Run(ctx, s, args)
	detail := name
	if err != nil {
		detail = name + " (error)"
	}
	if c != nil {
		s.audit(c, "ai_tool", detail)
	}
	return out, err
}
