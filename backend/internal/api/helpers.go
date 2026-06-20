package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

// currentUser returns the authenticated user's id, username and role.
func currentUser(c *gin.Context) (uint, string, string) {
	uid, _ := c.Get(ctxUserID)
	name, _ := c.Get(ctxUsername)
	role, _ := c.Get(ctxRole)
	id, _ := uid.(uint)
	username, _ := name.(string)
	r, _ := role.(string)
	return id, username, r
}

// audit records a security-relevant action. Failures are ignored so auditing
// never blocks the primary operation.
func (s *Server) audit(c *gin.Context, action, detail string) {
	id, username, _ := currentUser(c)
	_ = s.db.Create(&model.AuditLog{
		UserID:   id,
		Username: username,
		Action:   action,
		Detail:   detail,
		IP:       c.ClientIP(),
	}).Error
}

// badRequest writes a 400 with the given message.
func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

// serverError writes a 500 with the given error.
func serverError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
