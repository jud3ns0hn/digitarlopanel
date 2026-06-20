package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleDockerStatus(c *gin.Context) {
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
