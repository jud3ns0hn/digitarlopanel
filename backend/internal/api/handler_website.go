package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/nginx"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/php"
)

// writeSiteVHost renders and installs a website's nginx config, applying any
// certificate and PHP version recorded for it, then reloads nginx. It is the
// single source of truth for a site's vhost so SSL and PHP settings compose.
func (s *Server) writeSiteVHost(c *gin.Context, site model.Website) error {
	v := nginx.VHost{Domain: site.Domain, Root: site.Root, ProxyPass: site.ProxyPass}

	var cert model.Certificate
	if err := s.db.Where("domain = ?", site.Domain).First(&cert).Error; err == nil {
		v.CertPath = cert.CertPath
		v.KeyPath = cert.KeyPath
	}
	if site.PHPVersion != "" {
		v.PHPSocket = phpSocketFor(s.os.Family, site.PHPVersion)
	}

	if err := nginx.Write(s.os.Family, v); err != nil {
		return err
	}
	_, _ = s.service.Reload(c.Request.Context(), "nginx")
	return nil
}

// phpSocketFor resolves the php-fpm socket for a version, preferring a detected
// installed version and falling back to the conventional Debian path.
func phpSocketFor(family osinfo.Family, version string) string {
	for _, v := range php.ListVersions(family) {
		if v.Version == version {
			return v.Socket
		}
	}
	return php.DebianSocketPath(version)
}

// domainPattern validates a hostname to keep it out of shell/file contexts.
var domainPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]{0,251}[a-zA-Z0-9])?$`)

// proxyPattern validates a reverse-proxy upstream URL.
var proxyPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9.\-]+(:[0-9]{1,5})?(/[a-zA-Z0-9._~/\-]*)?$`)

func (s *Server) handleWebsiteList(c *gin.Context) {
	var sites []model.Website
	if err := s.db.Order("id desc").Find(&sites).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, sites)
}

type websiteCreateRequest struct {
	Domain    string `json:"domain" binding:"required"`
	Root      string `json:"root" binding:"required"`
	ProxyPass string `json:"proxy_pass"`
}

func (s *Server) handleWebsiteCreate(c *gin.Context) {
	var req websiteCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "domain and root required")
		return
	}
	if !domainPattern.MatchString(req.Domain) {
		badRequest(c, "invalid domain")
		return
	}
	root, err := resolvePath(s.cfg.FileRoot, req.Root)
	if err != nil {
		badRequest(c, "invalid root path")
		return
	}
	if req.ProxyPass != "" && !proxyPattern.MatchString(req.ProxyPass) {
		badRequest(c, "invalid proxy target (use http://host:port)")
		return
	}

	site := model.Website{Domain: req.Domain, Root: root, ProxyPass: req.ProxyPass, Enabled: true}
	if err := s.db.Create(&site).Error; err != nil {
		badRequest(c, "domain already exists or invalid")
		return
	}

	if err := s.writeSiteVHost(c, site); err != nil {
		// Roll back the DB record so state stays consistent.
		s.db.Delete(&site)
		serverError(c, err)
		return
	}

	s.audit(c, "website_create", req.Domain)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "website": site})
}

type proxyRequest struct {
	ProxyPass string `json:"proxy_pass"` // empty clears the proxy
}

// handleWebsiteProxy sets or clears a website's reverse-proxy upstream.
func (s *Server) handleWebsiteProxy(c *gin.Context) {
	var req proxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "proxy_pass required")
		return
	}
	if req.ProxyPass != "" && !proxyPattern.MatchString(req.ProxyPass) {
		badRequest(c, "invalid proxy target (use http://host:port)")
		return
	}
	var site model.Website
	if err := s.db.First(&site, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "website not found"})
		return
	}
	site.ProxyPass = req.ProxyPass
	if err := s.db.Save(&site).Error; err != nil {
		serverError(c, err)
		return
	}
	if err := s.writeSiteVHost(c, site); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "website_proxy", site.Domain+" -> "+req.ProxyPass)
	c.JSON(http.StatusOK, site)
}

func (s *Server) handleWebsiteToggle(c *gin.Context) {
	var site model.Website
	if err := s.db.First(&site, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "website not found"})
		return
	}
	site.Enabled = !site.Enabled
	if site.Enabled {
		_ = s.writeSiteVHost(c, site)
	} else {
		_ = nginx.Remove(s.os.Family, site.Domain)
		_, _ = s.service.Reload(c.Request.Context(), "nginx")
	}
	s.db.Save(&site)
	s.audit(c, "website_toggle", site.Domain)
	c.JSON(http.StatusOK, site)
}

func (s *Server) handleWebsiteDelete(c *gin.Context) {
	var site model.Website
	if err := s.db.First(&site, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "website not found"})
		return
	}
	_ = nginx.Remove(s.os.Family, site.Domain)
	_, _ = s.service.Reload(c.Request.Context(), "nginx")
	if err := s.db.Delete(&site).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "website_delete", site.Domain)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
