package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// handle2FASetup generates a fresh TOTP secret and returns the otpauth URL so
// the client can render a QR code. The secret is stored but not yet active.
func (s *Server) handle2FASetup(c *gin.Context) {
	id, username, _ := currentUser(c)
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		serverError(c, err)
		return
	}
	if user.TwoFAEnabled {
		badRequest(c, "two-factor authentication is already enabled")
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "DigitarloPanel",
		AccountName: username,
	})
	if err != nil {
		serverError(c, err)
		return
	}
	user.TwoFASecret = key.Secret()
	if err := s.db.Save(&user).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": key.Secret(), "otpauth_url": key.URL()})
}

type twoFAVerifyRequest struct {
	Code string `json:"code" binding:"required"`
}

// handle2FAEnable activates 2FA once the user proves possession of the secret.
func (s *Server) handle2FAEnable(c *gin.Context) {
	var req twoFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "code required")
		return
	}
	id, _, _ := currentUser(c)
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		serverError(c, err)
		return
	}
	if user.TwoFASecret == "" {
		badRequest(c, "run setup first")
		return
	}
	if !verifyTOTP(user.TwoFASecret, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid code"})
		return
	}
	user.TwoFAEnabled = true
	if err := s.db.Save(&user).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "2fa_enable", user.Username)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type twoFADisableRequest struct {
	Password string `json:"password" binding:"required"`
}

// handle2FADisable turns off 2FA after re-authenticating with the password.
func (s *Server) handle2FADisable(c *gin.Context) {
	var req twoFADisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "password required")
		return
	}
	id, _, _ := currentUser(c)
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		serverError(c, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
		return
	}
	user.TwoFAEnabled = false
	user.TwoFASecret = ""
	if err := s.db.Save(&user).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "2fa_disable", user.Username)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
