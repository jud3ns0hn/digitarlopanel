package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

// handleAppStoreList returns the curated catalog grouped by category, annotated
// with install status.
func (s *Server) handleAppStoreList(c *gin.Context) {
	var installed []model.ComposeApp
	s.db.Find(&installed)
	installedSet := map[string]bool{}
	for _, a := range installed {
		installedSet[a.Name] = true
	}

	type entry struct {
		storeApp
		Installed bool `json:"installed"`
	}
	byCategory := map[string][]entry{}
	categories := []string{}
	for _, app := range appCatalog {
		if _, ok := byCategory[app.Category]; !ok {
			categories = append(categories, app.Category)
		}
		byCategory[app.Category] = append(byCategory[app.Category], entry{storeApp: app, Installed: installedSet[app.Key]})
	}
	sort.Strings(categories)

	groups := make([]gin.H, 0, len(categories))
	for _, cat := range categories {
		groups = append(groups, gin.H{"category": cat, "apps": byCategory[cat]})
	}
	c.JSON(http.StatusOK, gin.H{"available": s.docker.Available(), "groups": groups})
}

type appStoreInstallRequest struct {
	Key string `json:"key" binding:"required"`
}

// handleAppStoreInstall deploys a catalog app via docker compose.
func (s *Server) handleAppStoreInstall(c *gin.Context) {
	if !s.docker.Available() {
		badRequest(c, "docker is not installed")
		return
	}
	var req appStoreInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "key required")
		return
	}
	app, ok := findStoreApp(req.Key)
	if !ok {
		badRequest(c, "unknown app")
		return
	}

	dir := filepath.Join(s.composeBaseDir(), app.Key)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		serverError(c, err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(app.Compose), 0o640); err != nil {
		serverError(c, err)
		return
	}

	out, err := s.docker.ComposeUp(c.Request.Context(), dir, app.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}

	rec := model.ComposeApp{Name: app.Key, Dir: dir}
	var existing model.ComposeApp
	if err := s.db.Where("name = ?", app.Key).First(&existing).Error; err == nil {
		rec.ID = existing.ID
		rec.CreatedAt = existing.CreatedAt
	}
	s.db.Save(&rec)
	s.audit(c, "appstore_install", app.Key)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out, "app": app})
}
