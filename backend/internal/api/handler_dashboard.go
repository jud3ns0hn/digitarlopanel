package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/system"
)

func (s *Server) handleHostInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"host": system.Host(c.Request.Context()),
		"os":   s.os,
	})
}

func (s *Server) handleProcesses(c *gin.Context) {
	procs, err := system.TopProcesses(c.Request.Context(), 20)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, procs)
}

type killProcessRequest struct {
	PID    int    `json:"pid" binding:"required"`
	Signal string `json:"signal"` // TERM (default) or KILL
}

// handleProcessKill sends a termination signal to a process. Admin-only and
// audited; the panel runs as root so this can stop any process.
func (s *Server) handleProcessKill(c *gin.Context) {
	var req killProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PID <= 1 {
		badRequest(c, "valid pid (> 1) required")
		return
	}
	sig := "-TERM"
	if req.Signal == "KILL" {
		sig = "-KILL"
	}
	if res, err := s.runner.Run(c.Request.Context(), "kill", sig, strconv.Itoa(req.PID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.CombinedOutput()})
		return
	}
	s.audit(c, "process_kill", sig+" "+strconv.Itoa(req.PID))
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleMetrics(c *gin.Context) {
	m, err := system.Collect(c.Request.Context(), 500*time.Millisecond)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

var wsUpgrader = websocket.Upgrader{
	// Same-origin is enforced by the bearer token; the panel is not a public API.
	CheckOrigin: func(*http.Request) bool { return true },
}

// handleMetricsStream pushes a metrics snapshot over WebSocket every 2 seconds.
func (s *Server) handleMetricsStream(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx := c.Request.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Send one immediately so the client does not wait for the first tick.
	send := func() bool {
		m, err := system.Collect(ctx, 300*time.Millisecond)
		if err != nil {
			return false
		}
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		return conn.WriteJSON(m) == nil
	}
	if !send() {
		return
	}

	// Detect client disconnect.
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				conn.Close()
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !send() {
				return
			}
		}
	}
}
