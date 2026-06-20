package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxRemoteDownload = 500 << 20 // 500 MiB

type remoteDownloadRequest struct {
	URL  string `json:"url" binding:"required"`
	Dest string `json:"dest" binding:"required"` // destination directory
}

// handleRemoteDownload fetches an http(s) URL server-side and stores it under
// the destination directory. Restricted to admins/operators (write tier).
func (s *Server) handleRemoteDownload(c *gin.Context) {
	var req remoteDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "url and dest required")
		return
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		badRequest(c, "only http and https URLs are allowed")
		return
	}
	dir, ok := s.resolve(c, req.Dest)
	if !ok {
		return
	}

	name := filepath.Base(req.URL)
	if name == "" || name == "/" || name == "." {
		name = "download"
	}
	dest := filepath.Join(dir, filepath.Base(name))
	if _, ok := s.resolve(c, dest); !ok {
		return
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(req.URL)
	if err != nil {
		badRequest(c, "fetch failed: "+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		badRequest(c, "remote returned status "+resp.Status)
		return
	}

	out, err := os.Create(dest)
	if err != nil {
		serverError(c, err)
		return
	}
	defer out.Close()

	written, err := io.Copy(out, io.LimitReader(resp.Body, maxRemoteDownload))
	if err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "file_download_url", req.URL+" -> "+dest)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "path": dest, "bytes": written})
}
