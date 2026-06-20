package api

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

// httpURLPattern validates an http(s) URL for uptime monitors and webhooks.
var httpURLPattern = regexp.MustCompile(`^https?://[^\s]{1,500}$`)

// --- Uptime monitors --------------------------------------------------------

func (s *Server) handleMonitorList(c *gin.Context) {
	var monitors []model.UptimeMonitor
	if err := s.db.Order("id desc").Find(&monitors).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, monitors)
}

type monitorCreateRequest struct {
	Name      string `json:"name" binding:"required"`
	URL       string `json:"url" binding:"required"`
	IntervalS int    `json:"interval_s"`
}

func (s *Server) handleMonitorCreate(c *gin.Context) {
	var req monitorCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpURLPattern.MatchString(req.URL) {
		badRequest(c, "name and valid http(s) url required")
		return
	}
	if req.IntervalS < 30 {
		req.IntervalS = 60
	}
	m := model.UptimeMonitor{Name: req.Name, URL: req.URL, IntervalS: req.IntervalS, Enabled: true}
	if err := s.db.Create(&m).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "monitor_create", req.Name)
	c.JSON(http.StatusOK, m)
}

func (s *Server) handleMonitorToggle(c *gin.Context) {
	var m model.UptimeMonitor
	if err := s.db.First(&m, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}
	m.Enabled = !m.Enabled
	s.db.Save(&m)
	c.JSON(http.StatusOK, m)
}

func (s *Server) handleMonitorDelete(c *gin.Context) {
	if err := s.db.Delete(&model.UptimeMonitor{}, c.Param("id")).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "monitor_delete", c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// --- Alert rules ------------------------------------------------------------

func (s *Server) handleAlertList(c *gin.Context) {
	var rules []model.AlertRule
	if err := s.db.Order("id desc").Find(&rules).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, rules)
}

type alertCreateRequest struct {
	Metric    string  `json:"metric" binding:"required"`
	Threshold float64 `json:"threshold" binding:"required"`
	Channel   string  `json:"channel" binding:"required"`
	Target    string  `json:"target" binding:"required"`
}

func (s *Server) handleAlertCreate(c *gin.Context) {
	var req alertCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid payload")
		return
	}
	switch req.Metric {
	case "cpu", "memory", "disk":
	default:
		badRequest(c, "metric must be cpu, memory or disk")
		return
	}
	if req.Threshold <= 0 || req.Threshold > 100 {
		badRequest(c, "threshold must be between 1 and 100")
		return
	}
	switch req.Channel {
	case "email":
		if !strings.Contains(req.Target, "@") {
			badRequest(c, "target must be an email address")
			return
		}
	case "webhook":
		if !httpURLPattern.MatchString(req.Target) {
			badRequest(c, "target must be an http(s) URL")
			return
		}
	default:
		badRequest(c, "channel must be email or webhook")
		return
	}
	rule := model.AlertRule{
		Metric: req.Metric, Threshold: req.Threshold,
		Channel: req.Channel, Target: req.Target, Enabled: true,
	}
	if err := s.db.Create(&rule).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "alert_create", req.Metric)
	c.JSON(http.StatusOK, rule)
}

func (s *Server) handleAlertToggle(c *gin.Context) {
	var rule model.AlertRule
	if err := s.db.First(&rule, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}
	rule.Enabled = !rule.Enabled
	s.db.Save(&rule)
	c.JSON(http.StatusOK, rule)
}

func (s *Server) handleAlertDelete(c *gin.Context) {
	if err := s.db.Delete(&model.AlertRule{}, c.Param("id")).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "alert_delete", c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// --- GPU --------------------------------------------------------------------

type gpuInfo struct {
	Name        string `json:"name"`
	MemoryTotal string `json:"memory_total"`
	MemoryUsed  string `json:"memory_used"`
	Utilization string `json:"utilization"`
	Temperature string `json:"temperature"`
}

// handleGPUList reports NVIDIA GPUs via nvidia-smi. Returns available:false when
// no NVIDIA tooling is present (the common case on hosts without a GPU).
func (s *Server) handleGPUList(c *gin.Context) {
	out, err := s.runner.Run(c.Request.Context(), "nvidia-smi",
		"--query-gpu=name,memory.total,memory.used,utilization.gpu,temperature.gpu",
		"--format=csv,noheader,nounits")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}
	gpus := []gpuInfo{}
	for _, line := range strings.Split(strings.TrimSpace(out.Stdout), "\n") {
		f := strings.Split(line, ",")
		if len(f) < 5 {
			continue
		}
		for i := range f {
			f[i] = strings.TrimSpace(f[i])
		}
		gpus = append(gpus, gpuInfo{
			Name: f[0], MemoryTotal: f[1] + " MiB", MemoryUsed: f[2] + " MiB",
			Utilization: f[3] + " %", Temperature: f[4] + " °C",
		})
	}
	c.JSON(http.StatusOK, gin.H{"available": true, "gpus": gpus})
}
