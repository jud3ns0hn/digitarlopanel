package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/cloudbackup"
)

func writeProbeFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("digitarlopanel connectivity test\n"), 0o600)
}

func removeProbeFile(path string) { _ = os.Remove(path) }

func (s *Server) handleDestinationList(c *gin.Context) {
	var dests []model.BackupDestination
	if err := s.db.Order("id desc").Find(&dests).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, dests)
}

type destinationCreateRequest struct {
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type" binding:"required"` // s3 | sftp | webdav
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
}

func (s *Server) handleDestinationCreate(c *gin.Context) {
	var req destinationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "name and type required")
		return
	}
	switch req.Type {
	case "s3", "sftp", "webdav":
	default:
		badRequest(c, "type must be s3, sftp or webdav")
		return
	}
	dest := model.BackupDestination{
		Name: req.Name, Type: req.Type, Endpoint: req.Endpoint,
		Bucket: req.Bucket, AccessKey: req.AccessKey,
		SecretKey: req.SecretKey, Region: req.Region,
	}
	if err := s.db.Create(&dest).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "destination_create", req.Name)
	c.JSON(http.StatusOK, dest)
}

func (s *Server) handleDestinationDelete(c *gin.Context) {
	if err := s.db.Delete(&model.BackupDestination{}, c.Param("id")).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "destination_delete", c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// toCloudDest maps the stored model to the cloudbackup transport struct.
func toCloudDest(d model.BackupDestination) cloudbackup.Destination {
	return cloudbackup.Destination{
		Type: d.Type, Endpoint: d.Endpoint, Bucket: d.Bucket,
		AccessKey: d.AccessKey, SecretKey: d.SecretKey, Region: d.Region,
	}
}

// handleDestinationTest uploads a tiny probe object to verify credentials and
// connectivity without requiring an existing backup.
func (s *Server) handleDestinationTest(c *gin.Context) {
	var dest model.BackupDestination
	if err := s.db.First(&dest, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "destination not found"})
		return
	}
	probe := filepath.Join(s.cfg.BackupDir, ".digitarlopanel-probe")
	if err := writeProbeFile(probe); err != nil {
		serverError(c, err)
		return
	}
	defer removeProbeFile(probe)

	if err := cloudbackup.Upload(c.Request.Context(), toCloudDest(dest), probe, "digitarlopanel-test.txt"); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type backupUploadRequest struct {
	DestinationID uint `json:"destination_id" binding:"required"`
}

// handleBackupUpload pushes an existing backup archive to a remote destination.
func (s *Server) handleBackupUpload(c *gin.Context) {
	var req backupUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "destination_id required")
		return
	}
	var rec model.Backup
	if err := s.db.First(&rec, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if _, err := resolvePath(s.cfg.BackupDir, rec.Path); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "backup path outside backup directory"})
		return
	}
	var dest model.BackupDestination
	if err := s.db.First(&dest, req.DestinationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "destination not found"})
		return
	}
	key := filepath.Base(rec.Path)
	if err := cloudbackup.Upload(c.Request.Context(), toCloudDest(dest), rec.Path, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, "backup_upload", rec.Name+" -> "+dest.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "key": key})
}
