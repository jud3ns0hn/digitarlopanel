package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

// MCP exposes the panel's safe tool registry to external agents (Claude Code,
// Claude Desktop, ...) over a minimal JSON-RPC 2.0 endpoint. Authentication is
// the panel's MCP token; the token holder operates at operator level (the
// registry contains no destructive tools).

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
	Params  struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"params"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handleMCP(c *gin.Context) {
	if !s.mcpAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MCP token"})
		return
	}
	var req jsonRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, rpcError(nil, -32700, "parse error"))
		return
	}

	// Notifications (no id) get no response body.
	if req.ID == nil && req.Method != "initialize" {
		c.Status(http.StatusAccepted)
		return
	}

	switch req.Method {
	case "initialize":
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{
			"protocolVersion": "2024-11-05",
			"capabilities":    gin.H{"tools": gin.H{}},
			"serverInfo":      gin.H{"name": "digitarlopanel", "version": "1.0"},
		}))
	case "ping":
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{}))
	case "tools/list":
		tools := make([]gin.H, 0)
		for _, t := range aiTools() {
			tools = append(tools, gin.H{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": t.Schema,
			})
		}
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{"tools": tools}))
	case "tools/call":
		out, err := s.executeAITool(c.Request.Context(), c, model.RoleOperator, req.Params.Name, req.Params.Arguments)
		isErr := err != nil
		if isErr {
			out = err.Error()
		}
		s.audit(c, "mcp_tool_call", req.Params.Name)
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{
			"content": []gin.H{{"type": "text", "text": out}},
			"isError": isErr,
		}))
	default:
		c.JSON(http.StatusOK, rpcError(req.ID, -32601, "method not found"))
	}
}

func (s *Server) mcpAuthorized(c *gin.Context) bool {
	if s.cfg.MCPToken == "" {
		return false
	}
	token := bearerToken(c)
	return token == s.cfg.MCPToken
}

func rpcResult(id any, result any) gin.H {
	return gin.H{"jsonrpc": "2.0", "id": id, "result": result}
}

func rpcError(id any, code int, message string) gin.H {
	return gin.H{"jsonrpc": "2.0", "id": id, "error": jsonRPCError{Code: code, Message: message}}
}
