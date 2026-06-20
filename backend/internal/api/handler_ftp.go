package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

var ftpUserPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)

func (s *Server) handleFTPList(c *gin.Context) {
	var accounts []model.FTPAccount
	if err := s.db.Order("id desc").Find(&accounts).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": s.ftp.Available(), "accounts": accounts})
}

type ftpCreateRequest struct {
	Username string `json:"username" binding:"required"`
	Home     string `json:"home" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) handleFTPCreate(c *gin.Context) {
	if !s.ftp.Available() {
		badRequest(c, "pure-ftpd (pure-pw) is not installed")
		return
	}
	var req ftpCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "username, home and password required")
		return
	}
	if !ftpUserPattern.MatchString(req.Username) {
		badRequest(c, "username must be 1-32 chars: letters, digits, . _ -")
		return
	}
	home, err := resolvePath(s.cfg.FileRoot, req.Home)
	if err != nil {
		badRequest(c, "invalid home path")
		return
	}
	if len(req.Password) < 8 {
		badRequest(c, "password must be at least 8 characters")
		return
	}

	if out, err := s.ftp.Create(c.Request.Context(), req.Username, home, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	acc := model.FTPAccount{Username: req.Username, Home: home}
	if err := s.db.Create(&acc).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "ftp_create", req.Username)
	c.JSON(http.StatusOK, acc)
}

type ftpPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

func (s *Server) handleFTPPassword(c *gin.Context) {
	if !s.ftp.Available() {
		badRequest(c, "pure-ftpd is not installed")
		return
	}
	var req ftpPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Password) < 8 {
		badRequest(c, "password (min 8 chars) required")
		return
	}
	var acc model.FTPAccount
	if err := s.db.First(&acc, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	if out, err := s.ftp.SetPassword(c.Request.Context(), acc.Username, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
		return
	}
	s.audit(c, "ftp_password", acc.Username)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleFTPDelete(c *gin.Context) {
	var acc model.FTPAccount
	if err := s.db.First(&acc, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	if s.ftp.Available() {
		if out, err := s.ftp.Delete(c.Request.Context(), acc.Username); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": out})
			return
		}
	}
	if err := s.db.Delete(&acc).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "ftp_delete", acc.Username)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
