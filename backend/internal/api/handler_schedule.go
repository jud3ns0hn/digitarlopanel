package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

func (s *Server) handleScheduleList(c *gin.Context) {
	var schedules []model.ScheduledBackup
	if err := s.db.Order("id desc").Find(&schedules).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, schedules)
}

type scheduleRequest struct {
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type" binding:"required"`
	Source    string `json:"source" binding:"required"`
	Schedule  string `json:"schedule" binding:"required"`
	Retention int    `json:"retention"`
}

func (s *Server) handleScheduleCreate(c *gin.Context) {
	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "name, type, source and schedule required")
		return
	}
	if !backupNamePattern.MatchString(req.Name) {
		badRequest(c, "name must be alphanumeric/._- and at most 64 chars")
		return
	}
	if req.Type != "files" && req.Type != "database" {
		badRequest(c, "type must be 'files' or 'database'")
		return
	}
	if !scheduleField.MatchString(req.Schedule) {
		badRequest(c, "invalid cron schedule")
		return
	}
	if req.Retention <= 0 {
		req.Retention = 7
	}
	sch := model.ScheduledBackup{
		Name: req.Name, Type: req.Type, Source: req.Source,
		Schedule: req.Schedule, Retention: req.Retention, Enabled: true,
	}
	if err := s.db.Create(&sch).Error; err != nil {
		serverError(c, err)
		return
	}
	s.scheduler.Reload()
	s.audit(c, "schedule_create", req.Name)
	c.JSON(http.StatusOK, sch)
}

func (s *Server) handleScheduleToggle(c *gin.Context) {
	var sch model.ScheduledBackup
	if err := s.db.First(&sch, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}
	sch.Enabled = !sch.Enabled
	s.db.Save(&sch)
	s.scheduler.Reload()
	s.audit(c, "schedule_toggle", sch.Name)
	c.JSON(http.StatusOK, sch)
}

func (s *Server) handleScheduleDelete(c *gin.Context) {
	var sch model.ScheduledBackup
	if err := s.db.First(&sch, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}
	if err := s.db.Delete(&sch).Error; err != nil {
		serverError(c, err)
		return
	}
	s.scheduler.Reload()
	s.audit(c, "schedule_delete", sch.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
