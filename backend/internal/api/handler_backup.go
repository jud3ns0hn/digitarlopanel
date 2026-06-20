package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

var backupNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,64}$`)

func (s *Server) handleBackupList(c *gin.Context) {
	var backups []model.Backup
	if err := s.db.Order("id desc").Find(&backups).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, backups)
}

type backupCreateRequest struct {
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required"`   // files | database
	Source string `json:"source" binding:"required"` // directory path or database name
}

func (s *Server) handleBackupCreate(c *gin.Context) {
	var req backupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "name, type and source required")
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
	rec, err := s.runBackup(c.Request.Context(), req.Name, req.Type, req.Source)
	if err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "backup_create", req.Type+":"+req.Source)
	c.JSON(http.StatusOK, rec)
}

// runBackup performs a backup and records it, independent of any HTTP request so
// it can be invoked by the scheduler. Callers validate name/type beforehand.
func (s *Server) runBackup(ctx context.Context, name, typ, source string) (model.Backup, error) {
	if err := os.MkdirAll(s.cfg.BackupDir, 0o750); err != nil {
		return model.Backup{}, err
	}
	stamp := time.Now().Format("20060102-150405")

	var (
		archive string
		err     error
	)
	switch typ {
	case "files":
		archive, err = s.backupFiles(ctx, name, stamp, source)
	case "database":
		archive, err = s.backupDatabase(ctx, name, stamp, source)
	default:
		return model.Backup{}, fmt.Errorf("invalid backup type: %s", typ)
	}
	if err != nil {
		return model.Backup{}, err
	}

	var size int64
	if info, statErr := os.Stat(archive); statErr == nil {
		size = info.Size()
	}
	rec := model.Backup{Name: name, Type: typ, Source: source, Path: archive, Size: size}
	if err := s.db.Create(&rec).Error; err != nil {
		return model.Backup{}, err
	}
	return rec, nil
}

func (s *Server) backupFiles(ctx context.Context, name, stamp, source string) (string, error) {
	src, err := resolvePath(s.cfg.FileRoot, source)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(src); err != nil {
		return "", fmt.Errorf("source not found: %w", err)
	}
	archive := filepath.Join(s.cfg.BackupDir, fmt.Sprintf("%s-%s.tar.gz", name, stamp))
	// tar receives the source via -C parent + base name; no shell interpolation.
	parent := filepath.Dir(src)
	base := filepath.Base(src)
	res, err := s.runner.Run(ctx, "tar", "-czf", archive, "-C", parent, base)
	if err != nil {
		return "", fmt.Errorf("tar: %s", res.CombinedOutput())
	}
	return archive, nil
}

func (s *Server) backupDatabase(ctx context.Context, name, stamp, dbName string) (string, error) {
	if !identPattern.MatchString(dbName) {
		return "", fmt.Errorf("invalid database name")
	}
	archive := filepath.Join(s.cfg.BackupDir, fmt.Sprintf("%s-%s.sql", name, stamp))
	res, err := s.runner.Run(ctx, "mysqldump", "--single-transaction", "--databases", dbName, "--result-file="+archive)
	if err != nil {
		return "", fmt.Errorf("mysqldump: %s", res.CombinedOutput())
	}
	return archive, nil
}

func (s *Server) handleBackupDownload(c *gin.Context) {
	var rec model.Backup
	if err := s.db.First(&rec, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	// Ensure the stored path is still within the backup directory.
	if _, err := resolvePath(s.cfg.BackupDir, rec.Path); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "backup path outside backup directory"})
		return
	}
	c.FileAttachment(rec.Path, filepath.Base(rec.Path))
}

func (s *Server) handleBackupDelete(c *gin.Context) {
	var rec model.Backup
	if err := s.db.First(&rec, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if _, err := resolvePath(s.cfg.BackupDir, rec.Path); err == nil {
		_ = os.Remove(rec.Path)
	}
	if err := s.db.Delete(&rec).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "backup_delete", rec.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
