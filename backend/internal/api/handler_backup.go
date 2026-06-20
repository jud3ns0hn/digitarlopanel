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
	if err := os.MkdirAll(s.cfg.BackupDir, 0o750); err != nil {
		serverError(c, err)
		return
	}

	stamp := time.Now().Format("20060102-150405")
	ctx := c.Request.Context()

	var (
		archive string
		err     error
	)
	switch req.Type {
	case "files":
		archive, err = s.backupFiles(ctx, req.Name, stamp, req.Source)
	case "database":
		archive, err = s.backupDatabase(ctx, req.Name, stamp, req.Source)
	default:
		badRequest(c, "type must be 'files' or 'database'")
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}

	info, _ := os.Stat(archive)
	var size int64
	if info != nil {
		size = info.Size()
	}
	rec := model.Backup{Name: req.Name, Type: req.Type, Source: req.Source, Path: archive, Size: size}
	if err := s.db.Create(&rec).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "backup_create", req.Type+":"+req.Source)
	c.JSON(http.StatusOK, rec)
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
