package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

// unitPattern restricts systemd unit names to safe characters so they cannot be
// used to inject extra arguments.
var unitPattern = regexp.MustCompile(`^[a-zA-Z0-9@._-]{1,128}$`)

func (s *Server) handleServiceList(c *gin.Context) {
	units, err := s.service.List(c.Request.Context())
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, units)
}

type unitActionRequest struct {
	Unit   string `json:"unit" binding:"required"`
	Action string `json:"action" binding:"required"`
}

func (s *Server) handleServiceControl(c *gin.Context) {
	var req unitActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "unit and action required")
		return
	}
	if !unitPattern.MatchString(req.Unit) {
		badRequest(c, "invalid unit name")
		return
	}
	ctx := c.Request.Context()
	var (
		out string
		err error
	)
	switch req.Action {
	case "start":
		out, err = s.service.Start(ctx, req.Unit)
	case "stop":
		out, err = s.service.Stop(ctx, req.Unit)
	case "restart":
		out, err = s.service.Restart(ctx, req.Unit)
	case "enable":
		out, err = s.service.Enable(ctx, req.Unit)
	case "disable":
		out, err = s.service.Disable(ctx, req.Unit)
	default:
		badRequest(c, "invalid action")
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "service_"+req.Action, req.Unit)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}
