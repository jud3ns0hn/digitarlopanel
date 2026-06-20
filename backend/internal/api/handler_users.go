package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,64}$`)

func validRole(role string) bool {
	switch role {
	case model.RoleAdmin, model.RoleOperator, model.RoleViewer:
		return true
	}
	return false
}

func (s *Server) handleUserList(c *gin.Context) {
	var users []model.User
	if err := s.db.Order("id").Find(&users).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, users)
}

type userCreateRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

func (s *Server) handleUserCreate(c *gin.Context) {
	var req userCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "username, password and role required")
		return
	}
	if !usernamePattern.MatchString(req.Username) {
		badRequest(c, "username must be 3-64 chars: letters, digits, . _ -")
		return
	}
	if !validRole(req.Role) {
		badRequest(c, "invalid role")
		return
	}
	if err := validatePasswordStrength(req.Password); err != nil {
		badRequest(c, err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		serverError(c, err)
		return
	}
	user := model.User{Username: req.Username, PasswordHash: string(hash), Role: req.Role}
	if err := s.db.Create(&user).Error; err != nil {
		badRequest(c, "username already exists")
		return
	}
	s.audit(c, "user_create", req.Username+" ("+req.Role+")")
	c.JSON(http.StatusOK, user)
}

type roleUpdateRequest struct {
	Role string `json:"role" binding:"required"`
}

func (s *Server) handleUserRole(c *gin.Context) {
	var req roleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "role required")
		return
	}
	if !validRole(req.Role) {
		badRequest(c, "invalid role")
		return
	}
	var user model.User
	if err := s.db.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	// Prevent removing the last admin.
	if user.Role == model.RoleAdmin && req.Role != model.RoleAdmin && s.adminCount() <= 1 {
		badRequest(c, "cannot demote the last administrator")
		return
	}
	user.Role = req.Role
	user.TokenVersion++ // re-evaluate privileges immediately
	if err := s.db.Save(&user).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "user_role", user.Username+" -> "+req.Role)
	c.JSON(http.StatusOK, user)
}

type passwordResetRequest struct {
	Password string `json:"password" binding:"required"`
}

func (s *Server) handleUserResetPassword(c *gin.Context) {
	var req passwordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "password required")
		return
	}
	if err := validatePasswordStrength(req.Password); err != nil {
		badRequest(c, err.Error())
		return
	}
	var user model.User
	if err := s.db.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		serverError(c, err)
		return
	}
	user.PasswordHash = string(hash)
	user.TokenVersion++ // log out the target user everywhere
	if err := s.db.Save(&user).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "user_reset_password", user.Username)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleUserDelete(c *gin.Context) {
	id, _, _ := currentUser(c)
	var user model.User
	if err := s.db.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.ID == id {
		badRequest(c, "you cannot delete your own account")
		return
	}
	if user.Role == model.RoleAdmin && s.adminCount() <= 1 {
		badRequest(c, "cannot delete the last administrator")
		return
	}
	if err := s.db.Delete(&user).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "user_delete", user.Username)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) adminCount() int64 {
	var n int64
	s.db.Model(&model.User{}).Where("role = ?", model.RoleAdmin).Count(&n)
	return n
}
