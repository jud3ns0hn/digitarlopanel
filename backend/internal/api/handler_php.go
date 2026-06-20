package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/php"
)

var phpVersionPattern = regexp.MustCompile(`^\d+\.\d+$`)

func (s *Server) handlePHPList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"family":   s.os.Family,
		"versions": php.ListVersions(s.os.Family),
	})
}

type phpInstallRequest struct {
	Version string `json:"version" binding:"required"`
}

func (s *Server) handlePHPInstall(c *gin.Context) {
	var req phpInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "version required")
		return
	}
	if !phpVersionPattern.MatchString(req.Version) {
		badRequest(c, "version must look like 8.3")
		return
	}
	pkg := php.PackageName(s.os.Family, req.Version)
	out, err := s.pkg.Install(c.Request.Context(), pkg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "php_install", req.Version)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}

type phpAssignRequest struct {
	Version string `json:"version"` // empty string disables PHP for the site
}

// handleWebsitePHP assigns (or clears) the PHP version for a website and
// rewrites its vhost.
func (s *Server) handleWebsitePHP(c *gin.Context) {
	var req phpAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "version required")
		return
	}
	if req.Version != "" && !phpVersionPattern.MatchString(req.Version) {
		badRequest(c, "version must look like 8.3 (or empty to disable)")
		return
	}
	var site model.Website
	if err := s.db.First(&site, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "website not found"})
		return
	}
	site.PHPVersion = req.Version
	if err := s.db.Save(&site).Error; err != nil {
		serverError(c, err)
		return
	}
	if err := s.writeSiteVHost(c.Request.Context(), site); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "website_php", site.Domain+" -> "+req.Version)
	c.JSON(http.StatusOK, site)
}
