package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// maxFailedAttempts triggers a temporary lock after this many bad passwords.
const maxFailedAttempts = 5

// lockDuration is how long an account stays locked after too many failures.
const lockDuration = 15 * time.Minute

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"os":     s.os,
	})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Code     string `json:"code"` // TOTP code, required when 2FA is enabled
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
		// Run a dummy hash compare to blunt username enumeration via timing.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinvalidinv"), []byte(req.Password))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "account temporarily locked, try again later"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		s.registerFailedLogin(c, &user)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Password is correct; enforce the second factor if enabled.
	if user.TwoFAEnabled {
		if req.Code == "" {
			c.JSON(http.StatusOK, gin.H{"two_factor_required": true})
			return
		}
		if !verifyTOTP(user.TwoFASecret, req.Code) {
			s.registerFailedLogin(c, &user)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid two-factor code"})
			return
		}
	}

	now := time.Now()
	user.FailedAttempts = 0
	user.LockedUntil = nil
	user.LastLoginAt = &now
	user.LastLoginIP = c.ClientIP()
	s.db.Save(&user)

	token, err := IssueToken(s.cfg.JWTSecret, user.ID, user.Username, user.Role, user.TokenVersion)
	if err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "login", user.Username)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": user.ID, "username": user.Username, "role": user.Role, "two_fa_enabled": user.TwoFAEnabled},
	})
}

// registerFailedLogin increments the failure counter and locks the account once
// the threshold is reached.
func (s *Server) registerFailedLogin(c *gin.Context, user *model.User) {
	user.FailedAttempts++
	if user.FailedAttempts >= maxFailedAttempts {
		until := time.Now().Add(lockDuration)
		user.LockedUntil = &until
		user.FailedAttempts = 0
		s.audit(c, "account_locked", user.Username)
	}
	s.db.Save(user)
}

func (s *Server) handleLogout(c *gin.Context) {
	id, username, _ := currentUser(c)
	// Bump the token version, invalidating every issued token for this user.
	s.db.Model(&model.User{}).Where("id = ?", id).
		UpdateColumn("token_version", gorm.Expr("token_version + 1"))
	s.audit(c, "logout", username)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleMe(c *gin.Context) {
	id, _, _ := currentUser(c)
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":             user.ID,
		"username":       user.Username,
		"role":           user.Role,
		"two_fa_enabled": user.TwoFAEnabled,
	})
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
	if err := validatePasswordStrength(req.NewPassword); err != nil {
		badRequest(c, err.Error())
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
	user.TokenVersion++ // revoke other sessions
	if err := s.db.Save(&user).Error; err != nil {
		serverError(c, err)
		return
	}
	// Re-issue a token so the current session stays valid.
	token, err := IssueToken(s.cfg.JWTSecret, user.ID, user.Username, user.Role, user.TokenVersion)
	if err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "change_password", "password updated")
	c.JSON(http.StatusOK, gin.H{"status": "ok", "token": token})
}

func (s *Server) handleAuditList(c *gin.Context) {
	var logs []model.AuditLog
	if err := s.db.Order("id desc").Limit(200).Find(&logs).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}
