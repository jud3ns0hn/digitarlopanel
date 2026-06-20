package api

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/robfig/cron/v3"
)

// Scheduler runs ScheduledBackup definitions in-process using cron expressions.
type Scheduler struct {
	server *Server
	mu     sync.Mutex
	cron   *cron.Cron
}

// NewScheduler creates a scheduler bound to the server.
func NewScheduler(s *Server) *Scheduler {
	return &Scheduler{server: s}
}

// Start loads the schedules and begins running them.
func (sc *Scheduler) Start() {
	sc.Reload()
}

// Reload rebuilds the cron registrations from the database. Safe to call after
// any change to scheduled backups.
func (sc *Scheduler) Reload() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.cron != nil {
		sc.cron.Stop()
	}
	sc.cron = cron.New()

	var schedules []model.ScheduledBackup
	if err := sc.server.db.Where("enabled = ?", true).Find(&schedules).Error; err != nil {
		log.Printf("scheduler: load schedules: %v", err)
		return
	}
	for _, sch := range schedules {
		sch := sch // capture
		if _, err := sc.cron.AddFunc(sch.Schedule, func() { sc.run(sch.ID) }); err != nil {
			log.Printf("scheduler: invalid schedule %q for %q: %v", sch.Schedule, sch.Name, err)
		}
	}
	// Daily Let's Encrypt renewal check at 03:30.
	_, _ = sc.cron.AddFunc("30 3 * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		sc.server.renewExpiringCerts(ctx)
	})
	sc.cron.Start()
}

// run executes a single scheduled backup and applies retention.
func (sc *Scheduler) run(id uint) {
	var sch model.ScheduledBackup
	if err := sc.server.db.First(&sch, id).Error; err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	now := time.Now()
	sch.LastRunAt = &now
	_, err := sc.server.runBackup(ctx, sch.Name, sch.Type, sch.Source)
	if err != nil {
		sch.LastStatus = "error: " + err.Error()
		log.Printf("scheduler: backup %q failed: %v", sch.Name, err)
	} else {
		sch.LastStatus = "ok"
		sc.applyRetention(sch)
	}
	sc.server.db.Save(&sch)
}

// applyRetention deletes the oldest backups for a schedule beyond its retention
// count, removing both the archive files and the records.
func (sc *Scheduler) applyRetention(sch model.ScheduledBackup) {
	if sch.Retention <= 0 {
		return
	}
	var backups []model.Backup
	if err := sc.server.db.Where("name = ? AND type = ?", sch.Name, sch.Type).
		Order("id desc").Find(&backups).Error; err != nil {
		return
	}
	for i, b := range backups {
		if i < sch.Retention {
			continue
		}
		if _, err := resolvePath(sc.server.cfg.BackupDir, b.Path); err == nil {
			_ = os.Remove(b.Path)
		}
		sc.server.db.Delete(&b)
	}
}
