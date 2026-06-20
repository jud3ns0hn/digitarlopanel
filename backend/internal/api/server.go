package api

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/config"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/acme"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/dockerctl"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/firewall"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/pkgmgr"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/runner"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/service"
	"gorm.io/gorm"
)

// Server bundles the dependencies shared by all HTTP handlers.
type Server struct {
	cfg          *config.Config
	db           *gorm.DB
	os           osinfo.Info
	runner       runner.Runner
	pkg          *pkgmgr.Manager
	service      *service.Controller
	firewall     *firewall.Manager
	docker       *dockerctl.Manager
	issuer       *acme.Issuer
	certDir      string
	scheduler    *Scheduler
	loginLimiter *rateLimiter
}

// NewServer wires up the server dependencies.
func NewServer(cfg *config.Config, db *gorm.DB) *Server {
	osi := osinfo.Detect()
	run := runner.Exec{}
	certDir := filepath.Join(cfg.DataDir, "certs")
	accountDir := filepath.Join(cfg.DataDir, "acme-accounts")
	s := &Server{
		cfg:          cfg,
		db:           db,
		os:           osi,
		runner:       run,
		pkg:          pkgmgr.New(osi.Family, run),
		service:      service.New(run),
		firewall:     firewall.New(run),
		docker:       dockerctl.New(run),
		issuer:       acme.NewIssuer(accountDir, certDir, false),
		certDir:      certDir,
		loginLimiter: newRateLimiter(10, time.Minute),
	}
	s.scheduler = NewScheduler(s)
	s.scheduler.Start()
	return s
}

// Handler builds the gin engine with all routes and the embedded SPA, served
// from the provided filesystem (the embedded frontend build).
func (s *Server) Handler(webFS fs.FS) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger(), securityHeaders())

	api := r.Group("/api")
	s.registerRoutes(api)

	s.registerSPA(r, webFS)
	return r
}

// registerSPA serves the embedded single-page app, falling back to index.html
// for client-side routes.
func (s *Server) registerSPA(r *gin.Engine, webFS fs.FS) {
	index, err := fs.ReadFile(webFS, "index.html")
	hasIndex := err == nil
	fileServer := http.FileServer(http.FS(webFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if _, err := fs.Stat(webFS, trimLeadingSlash(path)); err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		if hasIndex {
			c.Data(http.StatusOK, "text/html; charset=utf-8", index)
			return
		}
		c.String(http.StatusOK, "DigitarloPanel backend is running. Frontend build not embedded.")
	})
}

func trimLeadingSlash(p string) string {
	if len(p) > 0 && p[0] == '/' {
		return p[1:]
	}
	return p
}
