package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
)

// catalogApp describes an installable application with per-family package names.
type catalogApp struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Unit        string `json:"unit"` // systemd unit
	debianPkg   string
	rhelPkg     string
}

// catalog is the curated set of one-click installable apps.
var catalog = []catalogApp{
	{Key: "nginx", Name: "Nginx", Description: "Hochperformanter Webserver & Reverse-Proxy", Unit: "nginx", debianPkg: "nginx", rhelPkg: "nginx"},
	{Key: "mariadb", Name: "MariaDB", Description: "MySQL-kompatibler Datenbankserver", Unit: "mariadb", debianPkg: "mariadb-server", rhelPkg: "mariadb-server"},
	{Key: "redis", Name: "Redis", Description: "In-Memory Key-Value-Store / Cache", Unit: "redis", debianPkg: "redis-server", rhelPkg: "redis"},
	{Key: "php-fpm", Name: "PHP-FPM", Description: "PHP FastCGI Process Manager", Unit: "php-fpm", debianPkg: "php-fpm", rhelPkg: "php-fpm"},
}

func (a catalogApp) pkgFor(family osinfo.Family) string {
	if family == osinfo.FamilyDebian {
		return a.debianPkg
	}
	return a.rhelPkg
}

func (s *Server) findApp(key string) (catalogApp, bool) {
	for _, a := range catalog {
		if a.Key == key {
			return a, true
		}
	}
	return catalogApp{}, false
}

type softwareStatus struct {
	catalogApp
	Installed bool   `json:"installed"`
	Active    bool   `json:"active"`
	Enabled   bool   `json:"enabled"`
	State     string `json:"state"`
}

func (s *Server) handleSoftwareList(c *gin.Context) {
	ctx := c.Request.Context()
	out := make([]softwareStatus, 0, len(catalog))
	for _, app := range catalog {
		installed := s.pkg.IsInstalled(ctx, app.pkgFor(s.os.Family))
		st := s.service.Status(ctx, app.Unit)
		out = append(out, softwareStatus{
			catalogApp: app,
			Installed:  installed,
			Active:     st.Active,
			Enabled:    st.Enabled,
			State:      st.State,
		})
	}
	c.JSON(http.StatusOK, gin.H{"family": s.os.Family, "tool": s.pkg.Tool(), "apps": out})
}

type softwareRequest struct {
	Key string `json:"key" binding:"required"`
}

func (s *Server) handleSoftwareInstall(c *gin.Context) {
	var req softwareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "key required")
		return
	}
	app, ok := s.findApp(req.Key)
	if !ok {
		badRequest(c, "unknown application")
		return
	}
	out, err := s.pkg.Install(c.Request.Context(), app.pkgFor(s.os.Family))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "software_install", app.Key)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}

func (s *Server) handleSoftwareUninstall(c *gin.Context) {
	var req softwareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "key required")
		return
	}
	app, ok := s.findApp(req.Key)
	if !ok {
		badRequest(c, "unknown application")
		return
	}
	out, err := s.pkg.Remove(c.Request.Context(), app.pkgFor(s.os.Family))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "software_uninstall", app.Key)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}

type serviceActionRequest struct {
	Key    string `json:"key" binding:"required"`
	Action string `json:"action" binding:"required"` // start|stop|restart|enable|disable
}

func (s *Server) handleServiceAction(c *gin.Context) {
	var req serviceActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "key and action required")
		return
	}
	app, ok := s.findApp(req.Key)
	if !ok {
		badRequest(c, "unknown application")
		return
	}
	ctx := c.Request.Context()
	var (
		out string
		err error
	)
	switch req.Action {
	case "start":
		out, err = s.service.Start(ctx, app.Unit)
	case "stop":
		out, err = s.service.Stop(ctx, app.Unit)
	case "restart":
		out, err = s.service.Restart(ctx, app.Unit)
	case "enable":
		out, err = s.service.Enable(ctx, app.Unit)
	case "disable":
		out, err = s.service.Disable(ctx, app.Unit)
	default:
		badRequest(c, "invalid action")
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "service_"+req.Action, app.Key)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": out})
}
