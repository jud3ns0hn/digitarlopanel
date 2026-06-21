package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleDockerInstall installs the Docker Engine via Docker's official
// convenience script (works on Debian/Ubuntu and RHEL/Rocky/CentOS), enables
// the service and re-detects the CLI so Docker features work without a restart.
func (s *Server) handleDockerInstall(c *gin.Context) {
	ctx := c.Request.Context()
	if s.docker.Available() {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "already": true})
		return
	}
	// Fixed command string (no user input) piped into a shell.
	if res, err := s.runner.Run(ctx, "sh", "-c", "curl -fsSL https://get.docker.com | sh"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "docker install failed: " + res.CombinedOutput()})
		return
	}
	_, _ = s.service.Enable(ctx, "docker")
	_, _ = s.service.Start(ctx, "docker")
	available := s.docker.Recheck()
	s.audit(c, "docker_install", "")
	c.JSON(http.StatusOK, gin.H{"status": "ok", "available": available})
}

func (s *Server) handleDockerStatus(c *gin.Context) {
	// Re-detect each call so a Docker installed after startup is picked up live.
	if !s.docker.Available() {
		s.docker.Recheck()
	}
	if !s.docker.Available() {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}
	ctx := c.Request.Context()
	containers, err := s.docker.Containers(ctx)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": true, "error": err.Error()})
		return
	}
	images, _ := s.docker.Images(ctx)
	c.JSON(http.StatusOK, gin.H{"available": true, "containers": containers, "images": images})
}

// handleDockerInspect returns structured details for a single container.
func (s *Server) handleDockerInspect(c *gin.Context) {
	id := c.Query("id")
	detail, err := s.docker.Inspect(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// A live resource snapshot, best effort.
	stats, _ := s.docker.ContainerStats(c.Request.Context(), id)
	c.JSON(http.StatusOK, gin.H{"detail": detail, "stats": stats})
}

func (s *Server) handleDockerNetworks(c *gin.Context) {
	nets, err := s.docker.Networks(c.Request.Context())
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, nets)
}

func (s *Server) handleDockerVolumes(c *gin.Context) {
	vols, err := s.docker.Volumes(c.Request.Context())
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, vols)
}

func (s *Server) handleDockerLogs(c *gin.Context) {
	out, err := s.docker.ContainerLogs(c.Request.Context(), c.Query("id"), 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": out})
}

func (s *Server) handleDockerStats(c *gin.Context) {
	out, err := s.docker.ContainerStats(c.Request.Context(), c.Query("id"))
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"stats": out})
}

func (s *Server) handleDockerPrune(c *gin.Context) {
	out, err := s.docker.Prune(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "docker_prune", "")
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}

type dockerActionRequest struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"`
}

func (s *Server) handleDockerContainerAction(c *gin.Context) {
	if !s.docker.Available() {
		badRequest(c, "docker is not installed")
		return
	}
	var req dockerActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "id and action required")
		return
	}
	out, err := s.docker.ContainerAction(c.Request.Context(), req.ID, req.Action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "docker_"+req.Action, req.ID)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}

type dockerPullRequest struct {
	Image string `json:"image" binding:"required"`
}

func (s *Server) handleDockerPull(c *gin.Context) {
	if !s.docker.Available() {
		badRequest(c, "docker is not installed")
		return
	}
	var req dockerPullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "image required")
		return
	}
	out, err := s.docker.Pull(c.Request.Context(), req.Image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "docker_pull", req.Image)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}
