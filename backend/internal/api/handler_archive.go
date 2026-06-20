package api

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type compressRequest struct {
	Paths []string `json:"paths" binding:"required"` // items to include
	Dest  string   `json:"dest" binding:"required"`  // target .zip path
}

// handleFileCompress packs the given paths into a zip archive.
func (s *Server) handleFileCompress(c *gin.Context) {
	var req compressRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Paths) == 0 {
		badRequest(c, "paths and dest required")
		return
	}
	dest, ok := s.resolve(c, req.Dest)
	if !ok {
		return
	}
	out, err := os.Create(dest)
	if err != nil {
		serverError(c, err)
		return
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, p := range req.Paths {
		abs, err := resolvePath(s.cfg.FileRoot, p)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		base := filepath.Dir(abs)
		err = filepath.Walk(abs, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				_, err := zw.Create(rel + "/")
				return err
			}
			w, err := zw.Create(rel)
			if err != nil {
				return err
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(w, f)
			return err
		})
		if err != nil {
			serverError(c, err)
			return
		}
	}
	s.audit(c, "file_compress", dest)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "path": dest})
}

type extractRequest struct {
	Path string `json:"path" binding:"required"` // archive (.zip or .tar.gz)
	Dest string `json:"dest" binding:"required"` // destination directory
}

// handleFileExtract unpacks a .zip or .tar.gz archive into a directory, with
// protection against path traversal (zip-slip).
func (s *Server) handleFileExtract(c *gin.Context) {
	var req extractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "path and dest required")
		return
	}
	src, ok := s.resolve(c, req.Path)
	if !ok {
		return
	}
	dest, ok := s.resolve(c, req.Dest)
	if !ok {
		return
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		serverError(c, err)
		return
	}

	var err error
	switch {
	case strings.HasSuffix(src, ".zip"):
		err = extractZip(src, dest)
	case strings.HasSuffix(src, ".tar.gz"), strings.HasSuffix(src, ".tgz"):
		err = extractTarGz(src, dest)
	default:
		badRequest(c, "unsupported archive type (use .zip or .tar.gz)")
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "file_extract", src)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "dest": dest})
}

// safeJoin joins dest and name, rejecting paths that escape dest (zip-slip).
func safeJoin(dest, name string) (string, bool) {
	target := filepath.Join(dest, name)
	if target != dest && !strings.HasPrefix(target, dest+string(os.PathSeparator)) {
		return "", false
	}
	return target, true
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		target, ok := safeJoin(dest, f.Name)
		if !ok {
			continue // skip entries attempting traversal
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, ok := safeJoin(dest, hdr.Name)
		if !ok {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			// Limit copy to the declared size to bound resource use.
			if _, err := io.CopyN(out, tr, hdr.Size); err != nil && err != io.EOF {
				out.Close()
				return err
			}
			out.Close()
		}
	}
}
