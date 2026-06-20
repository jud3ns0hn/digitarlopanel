package api

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// logRoot constrains log file viewing regardless of the file-manager root.
const logRoot = "/var/log"

const defaultLogLines = 200
const maxLogLines = 2000

func clampLines(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLogLines
	}
	if n > maxLogLines {
		return maxLogLines
	}
	return n
}

// handleLogJournal returns the last N lines of a unit's journal.
func (s *Server) handleLogJournal(c *gin.Context) {
	unit := c.Query("unit")
	if !unitPattern.MatchString(unit) {
		badRequest(c, "invalid unit name")
		return
	}
	lines := clampLines(c.Query("lines"))
	res, err := s.runner.Run(c.Request.Context(), "journalctl",
		"-u", unit, "-n", strconv.Itoa(lines), "--no-pager", "--output", "short-iso")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": res.CombinedOutput()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unit": unit, "content": res.Stdout})
}

// handleLogFile returns the last N lines of a file under /var/log.
func (s *Server) handleLogFile(c *gin.Context) {
	abs, err := resolvePath(logRoot, c.Query("path"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "path must be under " + logRoot})
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		serverError(c, err)
		return
	}
	if info.IsDir() {
		entries, err := os.ReadDir(abs)
		if err != nil {
			serverError(c, err)
			return
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		c.JSON(http.StatusOK, gin.H{"path": abs, "is_dir": true, "entries": names})
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": abs, "content": tailLines(string(data), clampLines(c.Query("lines")))})
}

// tailLines returns the last n lines of s.
func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
