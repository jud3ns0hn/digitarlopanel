package api

import (
	"context"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/mem"
)

// timezonePattern matches IANA zone names like "Europe/Berlin" or "UTC".
var timezonePattern = regexp.MustCompile(`^[A-Za-z0-9+_-]+(/[A-Za-z0-9+_-]+){0,2}$`)

// swapNamePattern keeps the swapfile basename safe for argv/paths.
var swapNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,64}$`)

// handleToolboxSystem reports timezone, hostname and swap state in one call so
// the toolbox view can render without several round-trips.
func (s *Server) handleToolboxSystem(c *gin.Context) {
	ctx := c.Request.Context()
	tz, _ := s.runner.Run(ctx, "timedatectl", "show", "-p", "Timezone", "--value")
	host, _ := s.runner.Run(ctx, "hostnamectl", "hostname")

	resp := gin.H{
		"timezone": strings.TrimSpace(tz.CombinedOutput()),
		"hostname": strings.TrimSpace(host.CombinedOutput()),
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		resp["mem_total"] = vm.Total
	}
	if sw, err := mem.SwapMemory(); err == nil {
		resp["swap_total"] = sw.Total
		resp["swap_used"] = sw.Used
	}
	c.JSON(http.StatusOK, resp)
}

// handleToolboxTimezones lists the IANA timezones the host knows about.
func (s *Server) handleToolboxTimezones(c *gin.Context) {
	out, err := s.runner.Run(c.Request.Context(), "timedatectl", "list-timezones")
	if err != nil {
		serverError(c, err)
		return
	}
	zones := []string{}
	for _, z := range strings.Split(out.Stdout, "\n") {
		if z = strings.TrimSpace(z); z != "" {
			zones = append(zones, z)
		}
	}
	c.JSON(http.StatusOK, gin.H{"timezones": zones})
}

type timezoneRequest struct {
	Timezone string `json:"timezone" binding:"required"`
}

func (s *Server) handleToolboxSetTimezone(c *gin.Context) {
	var req timezoneRequest
	if err := c.ShouldBindJSON(&req); err != nil || !timezonePattern.MatchString(req.Timezone) {
		badRequest(c, "valid timezone required (e.g. Europe/Berlin)")
		return
	}
	if _, err := s.runner.Run(c.Request.Context(), "timedatectl", "set-timezone", req.Timezone); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "toolbox_timezone", req.Timezone)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "timezone": req.Timezone})
}

type hostnameRequest struct {
	Hostname string `json:"hostname" binding:"required"`
}

func (s *Server) handleToolboxSetHostname(c *gin.Context) {
	var req hostnameRequest
	if err := c.ShouldBindJSON(&req); err != nil || !domainPattern.MatchString(req.Hostname) {
		badRequest(c, "valid hostname required")
		return
	}
	if _, err := s.runner.Run(c.Request.Context(), "hostnamectl", "set-hostname", req.Hostname); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "toolbox_hostname", req.Hostname)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "hostname": req.Hostname})
}

type swapRequest struct {
	Name    string `json:"name"`     // file basename under /var, default "swapfile"
	SizeMB  int    `json:"size_mb"`  // size in MiB
	Disable bool   `json:"disable"`  // if true, swapoff + remove instead of create
}

// handleToolboxSwap creates (or removes) a swap file. Created files live under a
// fixed directory and are validated to keep paths and sizes sane.
func (s *Server) handleToolboxSwap(c *gin.Context) {
	var req swapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid payload")
		return
	}
	name := req.Name
	if name == "" {
		name = "swapfile"
	}
	if !swapNamePattern.MatchString(name) {
		badRequest(c, "invalid swap file name")
		return
	}
	path := filepath.Join("/var", name)
	ctx := c.Request.Context()

	if req.Disable {
		_, _ = s.runner.Run(ctx, "swapoff", path)
		if _, err := s.runner.Run(ctx, "rm", "-f", path); err != nil {
			serverError(c, err)
			return
		}
		s.audit(c, "toolbox_swap_disable", path)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	if req.SizeMB < 64 || req.SizeMB > 65536 {
		badRequest(c, "size_mb must be between 64 and 65536")
		return
	}
	steps := [][]string{
		{"fallocate", "-l", strconv.Itoa(req.SizeMB) + "M", path},
		{"chmod", "600", path},
		{"mkswap", path},
		{"swapon", path},
	}
	for _, step := range steps {
		if res, err := s.runner.Run(ctx, step[0], step[1:]...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": step[0] + ": " + res.CombinedOutput()})
			return
		}
	}
	s.audit(c, "toolbox_swap_create", path)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "path": path})
}

// sshSettings are the security-relevant sshd options the toolbox surfaces.
var sshKeys = []string{"Port", "PermitRootLogin", "PasswordAuthentication", "PubkeyAuthentication"}

// handleToolboxSSH reports effective sshd settings via `sshd -T`, which resolves
// defaults and includes values not explicitly set in the config file.
func (s *Server) handleToolboxSSH(c *gin.Context) {
	out, err := s.runner.Run(c.Request.Context(), "sshd", "-T")
	settings := gin.H{}
	if err == nil {
		lower := map[string]string{}
		for _, line := range strings.Split(out.Stdout, "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
			if len(parts) == 2 {
				lower[parts[0]] = parts[1]
			}
		}
		for _, k := range sshKeys {
			if v, ok := lower[strings.ToLower(k)]; ok {
				settings[k] = v
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"available": err == nil, "settings": settings})
}

// runToolboxAction is a small helper to enable+restart a unit after install.
func (s *Server) enableAndStart(ctx context.Context, unit string) {
	_, _ = s.service.Enable(ctx, unit)
	_, _ = s.service.Restart(ctx, unit)
}
