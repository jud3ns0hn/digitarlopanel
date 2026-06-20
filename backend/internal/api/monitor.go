package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/notify"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/system"
)

const (
	sampleInterval  = 30 * time.Second
	sampleRetention = 7 * 24 * time.Hour
	// alertCooldown suppresses repeat notifications for a firing rule.
	alertCooldown = 30 * time.Minute
	// uptimeTick is how often the uptime checker wakes to evaluate due monitors.
	uptimeTick = 15 * time.Second
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

	s.evaluateAlerts(ctx, sample)
}

// evaluateAlerts checks each enabled rule against the latest sample and fires a
// notification when a threshold is crossed (subject to a cooldown).
func (s *Server) evaluateAlerts(ctx context.Context, sample model.MetricSample) {
	var rules []model.AlertRule
	if err := s.db.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return
	}
	for _, rule := range rules {
		var value float64
		switch rule.Metric {
		case "cpu":
			value = sample.CPUPercent
		case "memory":
			value = sample.MemPercent
		case "disk":
			value = sample.DiskPercent
		default:
			continue
		}
		if value < rule.Threshold {
			continue
		}
		if rule.LastFired != nil && time.Since(*rule.LastFired) < alertCooldown {
			continue
		}
		s.fireAlert(ctx, rule, value)
		now := time.Now()
		rule.LastFired = &now
		s.db.Model(&rule).Update("last_fired", &now)
	}
}

// fireAlert dispatches a single alert over its configured channel.
func (s *Server) fireAlert(ctx context.Context, rule model.AlertRule, value float64) {
	subject := fmt.Sprintf("[DigitarloPanel] %s bei %.1f%% (Schwelle %.0f%%)", rule.Metric, value, rule.Threshold)
	body := fmt.Sprintf("Die Metrik %q liegt bei %.1f%% und hat die Schwelle von %.0f%% überschritten.\nZeit: %s",
		rule.Metric, value, rule.Threshold, time.Now().Format(time.RFC1123))

	var err error
	switch rule.Channel {
	case "email":
		err = notify.Email(s.smtpConfig(), rule.Target, subject, body)
	case "webhook":
		err = notify.Webhook(ctx, rule.Target, map[string]any{
			"metric": rule.Metric, "value": value, "threshold": rule.Threshold,
			"message": subject,
		})
	}
	if err != nil {
		log.Printf("alert: rule %d (%s) delivery failed: %v", rule.ID, rule.Channel, err)
	}
}

// smtpConfig adapts the panel config to the notify package.
func (s *Server) smtpConfig() notify.SMTPConfig {
	return notify.SMTPConfig{
		Host:     s.cfg.SMTPHost,
		Port:     s.cfg.SMTPPort,
		User:     s.cfg.SMTPUser,
		Password: s.cfg.SMTPPassword,
		From:     s.cfg.SMTPFrom,
	}
}

// startUptimeChecker periodically probes enabled monitors whose interval has
// elapsed, recording status, HTTP code and response time.
func (s *Server) startUptimeChecker() {
	go func() {
		ticker := time.NewTicker(uptimeTick)
		defer ticker.Stop()
		for range ticker.C {
			s.checkDueMonitors()
		}
	}()
}

func (s *Server) checkDueMonitors() {
	var monitors []model.UptimeMonitor
	if err := s.db.Where("enabled = ?", true).Find(&monitors).Error; err != nil {
		return
	}
	now := time.Now()
	for _, m := range monitors {
		if m.LastCheck != nil && now.Sub(*m.LastCheck) < time.Duration(m.IntervalS)*time.Second {
			continue
		}
		s.probeMonitor(m)
	}
}

func (s *Server) probeMonitor(m model.UptimeMonitor) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	status, code := "down", 0
	if err == nil {
		resp, derr := http.DefaultClient.Do(req)
		if derr == nil {
			code = resp.StatusCode
			resp.Body.Close()
			if code >= 200 && code < 400 {
				status = "up"
			}
		}
	}
	now := time.Now()
	s.db.Model(&m).Updates(map[string]any{
		"last_status": status,
		"last_code":   code,
		"last_ms":     now.Sub(start).Milliseconds(),
		"last_check":  &now,
	})
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
