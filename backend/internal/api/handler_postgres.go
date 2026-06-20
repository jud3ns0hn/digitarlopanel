package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

// psqlExec runs a SQL statement as the postgres superuser via peer auth.
func (s *Server) psqlExec(ctx context.Context, sql string) (string, error) {
	res, err := s.runner.Run(ctx, "runuser", "-u", "postgres", "--", "psql", "-tAc", sql)
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("psql: %w", err)
	}
	return out, nil
}

func (s *Server) handlePostgresList(c *gin.Context) {
	var dbs []model.PostgresInstance
	if err := s.db.Order("id desc").Find(&dbs).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, dbs)
}

type postgresCreateRequest struct {
	Name     string `json:"name" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password"`
}

func (s *Server) handlePostgresCreate(c *gin.Context) {
	var req postgresCreateRequest
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
	// Identifiers are validated; password is single-quoted (doubled quotes).
	stmt := fmt.Sprintf(
		`CREATE USER "%s" WITH PASSWORD '%s'; CREATE DATABASE "%s" OWNER "%s";`,
		req.Username, escapePGString(password), req.Name, req.Username,
	)
	if out, err := s.psqlExec(c.Request.Context(), stmt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	inst := model.PostgresInstance{Name: req.Name, Username: req.Username}
	if err := s.db.Create(&inst).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "postgres_create", req.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "database": inst, "password": password})
}

func (s *Server) handlePostgresDelete(c *gin.Context) {
	var inst model.PostgresInstance
	if err := s.db.First(&inst, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database not found"})
		return
	}
	stmt := fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"; DROP USER IF EXISTS "%s";`, inst.Name, inst.Username)
	if out, err := s.psqlExec(c.Request.Context(), stmt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	if err := s.db.Delete(&inst).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "postgres_delete", inst.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// escapePGString doubles single quotes for a single-quoted SQL literal.
func escapePGString(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\'' {
			out = append(out, '\'', '\'')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}
