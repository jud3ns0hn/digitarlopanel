package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

// maxIntegrityFiles caps how many files a single directory watch expands to,
// keeping baseline creation and scans bounded.
const maxIntegrityFiles = 2000

func (s *Server) handleIntegrityList(c *gin.Context) {
	var items []model.FileIntegrity
	if err := s.db.Order("path").Find(&items).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

type integrityAddRequest struct {
	Path string `json:"path" binding:"required"`
}

// handleIntegrityAdd registers a file (or every regular file under a directory)
// for tamper monitoring, recording each baseline hash.
func (s *Server) handleIntegrityAdd(c *gin.Context) {
	var req integrityAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "path required")
		return
	}
	root, err := resolvePath(s.cfg.FileRoot, req.Path)
	if err != nil {
		badRequest(c, "invalid path")
		return
	}
	info, err := os.Stat(root)
	if err != nil {
		badRequest(c, "path not found")
		return
	}

	var paths []string
	if info.IsDir() {
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !d.Type().IsRegular() {
				return nil
			}
			paths = append(paths, p)
			if len(paths) >= maxIntegrityFiles {
				return filepath.SkipAll
			}
			return nil
		})
	} else {
		paths = []string{root}
	}

	added := 0
	for _, p := range paths {
		hash, size, herr := hashFile(p)
		if herr != nil {
			continue
		}
		now := time.Now()
		rec := model.FileIntegrity{Path: p, Hash: hash, Size: size, Status: "ok", LastChecked: &now}
		// Upsert by unique path.
		var existing model.FileIntegrity
		if err := s.db.Where("path = ?", p).First(&existing).Error; err == nil {
			rec.ID = existing.ID
			rec.CreatedAt = existing.CreatedAt
		}
		if s.db.Save(&rec).Error == nil {
			added++
		}
	}
	s.audit(c, "integrity_add", root)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "tracked": added})
}

// handleIntegrityScan rehashes every tracked file and updates its status,
// returning the ones that changed or went missing.
func (s *Server) handleIntegrityScan(c *gin.Context) {
	var items []model.FileIntegrity
	if err := s.db.Find(&items).Error; err != nil {
		serverError(c, err)
		return
	}
	type change struct {
		Path   string `json:"path"`
		Status string `json:"status"`
	}
	changes := []change{}
	now := time.Now()
	for _, it := range items {
		status := "ok"
		hash, size, err := hashFile(it.Path)
		switch {
		case os.IsNotExist(err):
			status = "missing"
		case err != nil:
			status = "missing"
		case hash != it.Hash:
			status = "changed"
		}
		if status != "ok" {
			changes = append(changes, change{Path: it.Path, Status: status})
		}
		updates := map[string]any{"status": status, "last_checked": &now}
		if status == "ok" {
			updates["size"] = size
		}
		s.db.Model(&it).Updates(updates)
	}
	s.audit(c, "integrity_scan", "")
	c.JSON(http.StatusOK, gin.H{"checked": len(items), "changes": changes})
}

// handleIntegrityRebaseline resets a tracked file's baseline to its current
// content (use after a legitimate change).
func (s *Server) handleIntegrityRebaseline(c *gin.Context) {
	var it model.FileIntegrity
	if err := s.db.First(&it, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entry not found"})
		return
	}
	hash, size, err := hashFile(it.Path)
	if err != nil {
		badRequest(c, "file unreadable")
		return
	}
	now := time.Now()
	s.db.Model(&it).Updates(map[string]any{"hash": hash, "size": size, "status": "ok", "last_checked": &now})
	s.audit(c, "integrity_rebaseline", it.Path)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleIntegrityDelete(c *gin.Context) {
	if err := s.db.Delete(&model.FileIntegrity{}, c.Param("id")).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "integrity_delete", c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// hashFile returns the hex SHA-256 and size of a file.
func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
