package api

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

var composeNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

// composeBaseDir is where compose project directories are created.
func (s *Server) composeBaseDir() string {
	return filepath.Join(s.cfg.DataDir, "compose")
}

func (s *Server) handleComposeList(c *gin.Context) {
	if !s.docker.Available() {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}
	var apps []model.ComposeApp
	if err := s.db.Order("id desc").Find(&apps).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": true, "apps": apps})
}

type composeCreateRequest struct {
	Name    string `json:"name" binding:"required"`
	Content string `json:"content" binding:"required"` // docker-compose.yml content
}

func (s *Server) handleComposeDeploy(c *gin.Context) {
	if !s.docker.Available() {
		badRequest(c, "docker is not installed")
		return
	}
	var req composeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "name and content required")
		return
	}
	if !composeNamePattern.MatchString(req.Name) {
		badRequest(c, "name must be lowercase alphanumeric with - or _")
		return
	}

	dir := filepath.Join(s.composeBaseDir(), req.Name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		serverError(c, err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(req.Content), 0o640); err != nil {
		serverError(c, err)
		return
	}

	out, err := s.docker.ComposeUp(c.Request.Context(), dir, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}

	app := model.ComposeApp{Name: req.Name, Dir: dir}
	var existing model.ComposeApp
	if err := s.db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		app.ID = existing.ID
		app.CreatedAt = existing.CreatedAt
	}
	s.db.Save(&app)
	s.audit(c, "compose_deploy", req.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out, "app": app})
}

func (s *Server) handleComposeStatus(c *gin.Context) {
	var app model.ComposeApp
	if err := s.db.First(&app, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	out, _ := s.docker.ComposePs(c.Request.Context(), app.Dir, app.Name)
	c.JSON(http.StatusOK, gin.H{"name": app.Name, "status": out})
}

func (s *Server) handleComposeDown(c *gin.Context) {
	var app model.ComposeApp
	if err := s.db.First(&app, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	out, err := s.docker.ComposeDown(c.Request.Context(), app.Dir, app.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	if err := s.db.Delete(&app).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "compose_down", app.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}
