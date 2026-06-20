package api

import (
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// jailPattern matches fail2ban jail names (alphanumeric, dash, underscore).
var jailPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// supervisorNamePattern matches a supervised program or group:name.
var supervisorNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,128}$`)

// --- Fail2ban ---------------------------------------------------------------

func (s *Server) handleFail2banStatus(c *gin.Context) {
	out, err := s.runner.Run(c.Request.Context(), "fail2ban-client", "status")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}
	jails := []string{}
	for _, line := range strings.Split(out.Stdout, "\n") {
		if i := strings.Index(line, "Jail list:"); i >= 0 {
			list := strings.TrimSpace(line[i+len("Jail list:"):])
			for _, j := range strings.Split(list, ",") {
				if j = strings.TrimSpace(j); j != "" {
					jails = append(jails, j)
				}
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"available": true, "jails": jails})
}

func (s *Server) handleFail2banJail(c *gin.Context) {
	name := c.Query("name")
	if !jailPattern.MatchString(name) {
		badRequest(c, "invalid jail name")
		return
	}
	out, err := s.runner.Run(c.Request.Context(), "fail2ban-client", "status", name)
	if err != nil {
		serverError(c, err)
		return
	}
	banned := []string{}
	for _, line := range strings.Split(out.Stdout, "\n") {
		if i := strings.Index(line, "Banned IP list:"); i >= 0 {
			for _, ip := range strings.Fields(line[i+len("Banned IP list:"):]) {
				banned = append(banned, ip)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"jail": name, "raw": strings.TrimSpace(out.Stdout), "banned": banned})
}

type fail2banBanRequest struct {
	Jail string `json:"jail" binding:"required"`
	IP   string `json:"ip" binding:"required"`
}

func (s *Server) handleFail2banBan(c *gin.Context) { s.fail2banSet(c, "banip", "fail2ban_ban") }

func (s *Server) handleFail2banUnban(c *gin.Context) { s.fail2banSet(c, "unbanip", "fail2ban_unban") }

func (s *Server) fail2banSet(c *gin.Context, action, event string) {
	var req fail2banBanRequest
	if err := c.ShouldBindJSON(&req); err != nil || !jailPattern.MatchString(req.Jail) {
		badRequest(c, "jail and ip required")
		return
	}
	if net.ParseIP(req.IP) == nil {
		badRequest(c, "invalid IP address")
		return
	}
	if _, err := s.runner.Run(c.Request.Context(), "fail2ban-client", "set", req.Jail, action, req.IP); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, event, req.Jail+" "+req.IP)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleFail2banInstall(c *gin.Context) {
	ctx := c.Request.Context()
	if _, err := s.pkg.Install(ctx, "fail2ban"); err != nil {
		serverError(c, err)
		return
	}
	s.enableAndStart(ctx, "fail2ban")
	s.audit(c, "fail2ban_install", "")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// --- Supervisor -------------------------------------------------------------

type supervisorProc struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func (s *Server) handleSupervisorStatus(c *gin.Context) {
	out, err := s.runner.Run(c.Request.Context(), "supervisorctl", "status")
	// supervisorctl status exits non-zero when some processes are not RUNNING,
	// so parse output regardless of the error as long as we got lines.
	if strings.TrimSpace(out.Stdout) == "" && err != nil {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}
	procs := []supervisorProc{}
	for _, line := range strings.Split(out.Stdout, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		procs = append(procs, supervisorProc{
			Name:   f[0],
			Status: f[1],
			Detail: strings.TrimSpace(strings.Join(f[2:], " ")),
		})
	}
	c.JSON(http.StatusOK, gin.H{"available": true, "processes": procs})
}

type supervisorActionRequest struct {
	Name   string `json:"name" binding:"required"`
	Action string `json:"action" binding:"required"` // start|stop|restart
}

func (s *Server) handleSupervisorAction(c *gin.Context) {
	var req supervisorActionRequest
	if err := c.ShouldBindJSON(&req); err != nil || !supervisorNamePattern.MatchString(req.Name) {
		badRequest(c, "name and action required")
		return
	}
	switch req.Action {
	case "start", "stop", "restart":
	default:
		badRequest(c, "action must be start, stop or restart")
		return
	}
	out, err := s.runner.Run(c.Request.Context(), "supervisorctl", req.Action, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": out.CombinedOutput()})
		return
	}
	s.audit(c, "supervisor_"+req.Action, req.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": strings.TrimSpace(out.CombinedOutput())})
}

func (s *Server) handleSupervisorInstall(c *gin.Context) {
	ctx := c.Request.Context()
	if _, err := s.pkg.Install(ctx, "supervisor"); err != nil {
		serverError(c, err)
		return
	}
	s.enableAndStart(ctx, "supervisor")
	s.audit(c, "supervisor_install", "")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// --- ClamAV -----------------------------------------------------------------

func (s *Server) handleClamAVStatus(c *gin.Context) {
	out, err := s.runner.Run(c.Request.Context(), "clamscan", "--version")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": true, "version": strings.TrimSpace(out.CombinedOutput())})
}

type clamScanRequest struct {
	Path string `json:"path" binding:"required"`
}

// handleClamAVScan runs a recursive scan over a path inside the managed file
// root and returns the infected files. The scan is bounded by the runner's
// command timeout.
func (s *Server) handleClamAVScan(c *gin.Context) {
	var req clamScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "path required")
		return
	}
	target, err := resolvePath(s.cfg.FileRoot, req.Path)
	if err != nil {
		badRequest(c, "invalid path")
		return
	}
	out, _ := s.runner.Run(c.Request.Context(), "clamscan", "-r", "--infected", "--no-summary", target)
	infected := []string{}
	for _, line := range strings.Split(out.Stdout, "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), "FOUND") {
			infected = append(infected, strings.TrimSpace(line))
		}
	}
	s.audit(c, "clamav_scan", target)
	c.JSON(http.StatusOK, gin.H{"path": target, "infected": infected, "clean": len(infected) == 0})
}

func (s *Server) handleClamAVInstall(c *gin.Context) {
	ctx := c.Request.Context()
	if _, err := s.pkg.Install(ctx, "clamav"); err != nil {
		serverError(c, err)
		return
	}
	// Refresh signatures; ignore errors (freshclam may already run as a service).
	_, _ = s.runner.Run(ctx, "freshclam")
	s.audit(c, "clamav_install", "")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
