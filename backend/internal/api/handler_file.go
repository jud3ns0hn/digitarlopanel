package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

const maxEditableSize = 4 << 20 // 4 MiB cap for the text editor

type fileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime int64  `json:"mod_time"`
}

func (s *Server) resolve(c *gin.Context, p string) (string, bool) {
	abs, err := resolvePath(s.cfg.FileRoot, p)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return "", false
	}
	return abs, true
}

func (s *Server) handleFileList(c *gin.Context) {
	abs, ok := s.resolve(c, c.Query("path"))
	if !ok {
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		serverError(c, err)
		return
	}
	out := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, fileEntry{
			Name:    e.Name(),
			Path:    filepath.Join(abs, e.Name()),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			Mode:    fmt.Sprintf("%04o", info.Mode().Perm()),
			ModTime: info.ModTime().Unix(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"path": abs, "entries": out})
}

func (s *Server) handleFileRead(c *gin.Context) {
	abs, ok := s.resolve(c, c.Query("path"))
	if !ok {
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		serverError(c, err)
		return
	}
	if info.IsDir() {
		badRequest(c, "path is a directory")
		return
	}
	if info.Size() > maxEditableSize {
		badRequest(c, "file too large to edit in the browser")
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": abs, "content": string(data)})
}

type fileWriteRequest struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content"`
}

func (s *Server) handleFileWrite(c *gin.Context) {
	var req fileWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "path required")
		return
	}
	abs, ok := s.resolve(c, req.Path)
	if !ok {
		return
	}
	if err := os.WriteFile(abs, []byte(req.Content), 0o644); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "file_write", abs)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type pathRequest struct {
	Path string `json:"path" binding:"required"`
}

func (s *Server) handleFileMkdir(c *gin.Context) {
	var req pathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "path required")
		return
	}
	abs, ok := s.resolve(c, req.Path)
	if !ok {
		return
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "file_mkdir", abs)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type renameRequest struct {
	From string `json:"from" binding:"required"`
	To   string `json:"to" binding:"required"`
}

func (s *Server) handleFileRename(c *gin.Context) {
	var req renameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "from and to required")
		return
	}
	from, ok := s.resolve(c, req.From)
	if !ok {
		return
	}
	to, ok := s.resolve(c, req.To)
	if !ok {
		return
	}
	if err := os.Rename(from, to); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "file_rename", from+" -> "+to)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleFileDelete(c *gin.Context) {
	var req pathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "path required")
		return
	}
	abs, ok := s.resolve(c, req.Path)
	if !ok {
		return
	}
	if abs == "/" || abs == filepath.Clean(s.cfg.FileRoot) {
		badRequest(c, "refusing to delete the root directory")
		return
	}
	if err := os.RemoveAll(abs); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "file_delete", abs)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type chmodRequest struct {
	Path string `json:"path" binding:"required"`
	Mode string `json:"mode" binding:"required"` // octal, e.g. "0755"
}

func (s *Server) handleFileChmod(c *gin.Context) {
	var req chmodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "path and mode required")
		return
	}
	abs, ok := s.resolve(c, req.Path)
	if !ok {
		return
	}
	mode, err := strconv.ParseUint(req.Mode, 8, 32)
	if err != nil {
		badRequest(c, "mode must be octal, e.g. 0755")
		return
	}
	if err := os.Chmod(abs, os.FileMode(mode)); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "file_chmod", fmt.Sprintf("%s %s", abs, req.Mode))
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleFileDownload(c *gin.Context) {
	abs, ok := s.resolve(c, c.Query("path"))
	if !ok {
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		serverError(c, err)
		return
	}
	if info.IsDir() {
		badRequest(c, "cannot download a directory")
		return
	}
	c.FileAttachment(abs, filepath.Base(abs))
}

func (s *Server) handleFileUpload(c *gin.Context) {
	dir, ok := s.resolve(c, c.PostForm("path"))
	if !ok {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "file field required")
		return
	}
	dest := filepath.Join(dir, filepath.Base(file.Filename))
	if _, ok := s.resolve(c, dest); !ok {
		return
	}
	if err := c.SaveUploadedFile(file, dest); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "file_upload", dest)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "path": dest})
}
