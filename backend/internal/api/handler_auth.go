package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"os":     s.os,
	})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) handleLogin(c *gin.Context) {
	if !s.loginLimiter.allow(c.ClientIP()) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts, slow down"})
		return
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "username and password required")
		return
	}

	var user model.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := IssueToken(s.cfg.JWTSecret, user.ID, user.Username, user.Role)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": user.ID, "username": user.Username, "role": user.Role},
	})
}

func (s *Server) handleMe(c *gin.Context) {
	id, username, role := currentUser(c)
	c.JSON(http.StatusOK, gin.H{"id": id, "username": username, "role": role})
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (s *Server) handleChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "old_password and new_password required")
		return
	}
	if len(req.NewPassword) < 8 {
		badRequest(c, "new password must be at least 8 characters")
		return
	}

	id, _, _ := currentUser(c)
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		serverError(c, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		serverError(c, err)
		return
	}
	user.PasswordHash = string(hash)
	if err := s.db.Save(&user).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "change_password", "password updated")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleAuditList(c *gin.Context) {
	var logs []model.AuditLog
	if err := s.db.Order("id desc").Limit(200).Find(&logs).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}
