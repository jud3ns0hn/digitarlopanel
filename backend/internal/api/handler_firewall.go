package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleFirewallStatus(c *gin.Context) {
	status, err := s.firewall.Status(c.Request.Context())
	rules, _ := s.firewall.List(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"backend": s.firewall.Backend(),
		"status":  status,
		"rules":   rules,
		"error":   errString(err),
	})
}

type firewallRuleRequest struct {
	Port  int    `json:"port" binding:"required"`
	Proto string `json:"proto" binding:"required"`
}

func (s *Server) handleFirewallAllow(c *gin.Context) {
	var req firewallRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "port and proto required")
		return
	}
	out, err := s.firewall.Allow(c.Request.Context(), req.Port, req.Proto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "firewall_allow", out)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}

func (s *Server) handleFirewallDeny(c *gin.Context) {
	var req firewallRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "port and proto required")
		return
	}
	out, err := s.firewall.Deny(c.Request.Context(), req.Port, req.Proto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "firewall_deny", out)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
