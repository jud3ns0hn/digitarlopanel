package api

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

// crondPath is the managed drop-in file written into the system cron directory.
const crondPath = "/etc/cron.d/digitarlopanel"

// scheduleField rejects newlines and other characters that could break out of
// a crontab line.
var scheduleField = regexp.MustCompile(`^[0-9*,/\- A-Za-z]{1,128}$`)

func (s *Server) handleCronList(c *gin.Context) {
	var jobs []model.CronJob
	if err := s.db.Order("id desc").Find(&jobs).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, jobs)
}

type cronCreateRequest struct {
	Name     string `json:"name" binding:"required"`
	Schedule string `json:"schedule" binding:"required"`
	Command  string `json:"command" binding:"required"`
}

func (s *Server) handleCronCreate(c *gin.Context) {
	var req cronCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "name, schedule and command required")
		return
	}
	if !scheduleField.MatchString(req.Schedule) {
		badRequest(c, "invalid cron schedule")
		return
	}
	if strings.ContainsAny(req.Command, "\n\r") {
		badRequest(c, "command must not contain newlines")
		return
	}
	job := model.CronJob{Name: req.Name, Schedule: req.Schedule, Command: req.Command, Enabled: true}
	if err := s.db.Create(&job).Error; err != nil {
		serverError(c, err)
		return
	}
	if err := s.syncCrontab(); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "cron_create", req.Name)
	c.JSON(http.StatusOK, job)
}

func (s *Server) handleCronToggle(c *gin.Context) {
	var job model.CronJob
	if err := s.db.First(&job, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cron job not found"})
		return
	}
	job.Enabled = !job.Enabled
	s.db.Save(&job)
	if err := s.syncCrontab(); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "cron_toggle", job.Name)
	c.JSON(http.StatusOK, job)
}

func (s *Server) handleCronDelete(c *gin.Context) {
	var job model.CronJob
	if err := s.db.First(&job, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cron job not found"})
		return
	}
	if err := s.db.Delete(&job).Error; err != nil {
		serverError(c, err)
		return
	}
	if err := s.syncCrontab(); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "cron_delete", job.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// syncCrontab rewrites the managed cron.d drop-in from all enabled jobs. It is
// a no-op error if /etc/cron.d is not writable (e.g. when running unprivileged
// in development), so the panel keeps working.
func (s *Server) syncCrontab() error {
	var jobs []model.CronJob
	if err := s.db.Where("enabled = ?", true).Order("id").Find(&jobs).Error; err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Managed by DigitarloPanel - do not edit by hand\n")
	b.WriteString("SHELL=/bin/bash\n")
	b.WriteString("PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\n\n")
	for _, j := range jobs {
		fmt.Fprintf(&b, "# %s\n%s root %s\n", j.Name, j.Schedule, j.Command)
	}
	if err := os.WriteFile(crondPath, []byte(b.String()), 0o644); err != nil {
		if os.IsPermission(err) || os.IsNotExist(err) {
			// Development / unprivileged environments: skip silently.
			return nil
		}
		return err
	}
	return nil
}
