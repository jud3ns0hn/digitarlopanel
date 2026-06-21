package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func splitTabs(s string) []string { return strings.Split(s, "\t") }

// identPattern restricts database and user names to safe identifiers so they
// can be interpolated into SQL without injection risk.
var identPattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)

// mysqlExec runs a SQL statement via the local mysql client as root (socket auth).
func (s *Server) mysqlExec(ctx context.Context, sql string) (string, error) {
	res, err := s.runner.Run(ctx, "mysql", "-e", sql)
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("mysql: %w", err)
	}
	return out, nil
}

func (s *Server) handleDatabaseList(c *gin.Context) {
	var dbs []model.DatabaseInstance
	if err := s.db.Order("id desc").Find(&dbs).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, dbs)
}

type dbTable struct {
	Name   string `json:"name"`
	Rows   string `json:"rows"`
	SizeMB string `json:"size_mb"`
}

// handleDatabaseTables lists the tables of a database with approximate row
// counts and on-disk size (from information_schema). Read-only.
func (s *Server) handleDatabaseTables(c *gin.Context) {
	name := c.Query("name")
	if !identPattern.MatchString(name) {
		badRequest(c, "invalid database name")
		return
	}
	// information_schema is queried with the name single-quoted (already an
	// identifier, so doubly safe). Tab-separated, no header.
	sql := fmt.Sprintf(
		"SELECT table_name, table_rows, ROUND(((data_length+index_length)/1024/1024),2) "+
			"FROM information_schema.tables WHERE table_schema='%s' ORDER BY (data_length+index_length) DESC;",
		escapeSQLString(name))
	out, err := s.runner.Run(c.Request.Context(), "mysql", "-N", "-B", "-e", sql)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": out.CombinedOutput()})
		return
	}
	tables := []dbTable{}
	for _, line := range splitLines(out.Stdout) {
		f := splitTabs(line)
		if len(f) >= 3 {
			tables = append(tables, dbTable{Name: f[0], Rows: f[1], SizeMB: f[2]})
		}
	}
	c.JSON(http.StatusOK, gin.H{"database": name, "tables": tables})
}

type databaseCreateRequest struct {
	Name     string `json:"name" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password"`
}

func (s *Server) handleDatabaseCreate(c *gin.Context) {
	var req databaseCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "name and username required")
		return
	}
	if !identPattern.MatchString(req.Name) || !identPattern.MatchString(req.Username) {
		badRequest(c, "name and username must be alphanumeric/underscore, max 64 chars")
		return
	}
	password := req.Password
	if password == "" {
		var err error
		if password, err = randomDBPassword(20); err != nil {
			serverError(c, err)
			return
		}
	}

	// Names are validated identifiers; password is single-quoted by mysql.
	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4; "+
			"CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'; "+
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'; FLUSH PRIVILEGES;",
		req.Name, req.Username, escapeSQLString(password), req.Name, req.Username,
	)
	if out, err := s.mysqlExec(c.Request.Context(), stmt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}

	inst := model.DatabaseInstance{Name: req.Name, Username: req.Username, Charset: "utf8mb4"}
	if err := s.db.Create(&inst).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "database_create", req.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "database": inst, "password": password})
}

func (s *Server) handleDatabaseDelete(c *gin.Context) {
	var inst model.DatabaseInstance
	if err := s.db.First(&inst, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database not found"})
		return
	}
	stmt := fmt.Sprintf(
		"DROP DATABASE IF EXISTS `%s`; DROP USER IF EXISTS '%s'@'localhost'; FLUSH PRIVILEGES;",
		inst.Name, inst.Username,
	)
	if out, err := s.mysqlExec(c.Request.Context(), stmt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	if err := s.db.Delete(&inst).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "database_delete", inst.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// escapeSQLString escapes characters significant inside a single-quoted SQL string.
func escapeSQLString(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case '\'', '\\':
			out = append(out, '\\', r)
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

const dbPasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

func randomDBPassword(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = dbPasswordAlphabet[int(b)%len(dbPasswordAlphabet)]
	}
	return string(out), nil
}
