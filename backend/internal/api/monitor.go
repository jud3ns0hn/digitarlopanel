package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/system"
)

const (
	sampleInterval  = 30 * time.Second
	sampleRetention = 7 * 24 * time.Hour
)

// startMonitorSampler records a metric sample on a fixed interval and prunes old
// samples, so the dashboard can show history over time.
func (s *Server) startMonitorSampler() {
	go func() {
		ticker := time.NewTicker(sampleInterval)
		defer ticker.Stop()
		for range ticker.C {
			s.recordSample()
		}
	}()
}

func (s *Server) recordSample() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m, err := system.Collect(ctx, time.Second)
	if err != nil {
		return
	}
	var diskPercent float64
	for _, d := range m.Disks {
		if d.UsedPercent > diskPercent {
			diskPercent = d.UsedPercent
		}
	}
	sample := model.MetricSample{
		Timestamp:   m.Timestamp,
		CPUPercent:  m.CPUPercent,
		MemPercent:  m.Memory.UsedPercent,
		Load1:       m.Load1,
		DiskPercent: diskPercent,
	}
	s.db.Create(&sample)
	// Prune anything older than the retention window.
	cutoff := time.Now().Add(-sampleRetention).Unix()
	s.db.Where("timestamp < ?", cutoff).Delete(&model.MetricSample{})
}

// handleMetricsHistory returns samples within the requested window (seconds).
func (s *Server) handleMetricsHistory(c *gin.Context) {
	window := 3600
	if v, err := strconv.Atoi(c.Query("window")); err == nil && v > 0 && v <= int(sampleRetention.Seconds()) {
		window = v
	}
	since := time.Now().Add(-time.Duration(window) * time.Second).Unix()
	var samples []model.MetricSample
	if err := s.db.Where("timestamp >= ?", since).Order("timestamp").Find(&samples).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, samples)
}
