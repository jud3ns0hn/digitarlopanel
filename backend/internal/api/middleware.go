package api

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

const (
	ctxUserID   = "uid"
	ctxUsername = "username"
	ctxRole     = "role"
)

// requestLogger logs each request with method, path, status and latency.
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path,
			c.Writer.Status(), time.Since(start).Round(time.Millisecond))
	}
}

// authRequired validates the bearer token (or ?token= query for WebSocket and
// download endpoints), confirms the token has not been revoked by checking the
// user's current TokenVersion, and stores the fresh identity in the context.
func (s *Server) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := ParseToken(s.cfg.JWTSecret, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		var user model.User
		if err := s.db.First(&user, claims.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "account not found"})
			return
		}
		if user.TokenVersion != claims.TokenVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired, please log in again"})
			return
		}

		c.Set(ctxUserID, user.ID)
		c.Set(ctxUsername, user.Username)
		c.Set(ctxRole, user.Role) // fresh role, so privilege changes take effect immediately
		c.Next()
	}
}

// requireRole aborts the request unless the authenticated user holds one of the
// allowed roles. Admin always passes.
func (s *Server) requireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		_, _, role := currentUser(c)
		if role == model.RoleAdmin || allowed[role] {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
	}
}

// writeRole permits admin and operator (anything that mutates system state).
func (s *Server) writeRole() gin.HandlerFunc {
	return s.requireRole(model.RoleOperator)
}

// adminOnly permits admins exclusively.
func (s *Server) adminOnly() gin.HandlerFunc {
	return s.requireRole() // empty allow-list => only admin passes
}

// securityHeaders sets defensive HTTP response headers, including a strict
// Content-Security-Policy. Inline styles are permitted because Element Plus
// injects them at runtime; scripts are restricted to same-origin.
func securityHeaders() gin.HandlerFunc {
	const csp = "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; " +
		"font-src 'self' data:; " +
		"connect-src 'self' ws: wss:; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"frame-ancestors 'none'"
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		if c.Request.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if after, ok := strings.CutPrefix(header, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	// WebSocket and direct download links cannot set headers easily.
	return c.Query("token")
}

// rateLimiter is a simple fixed-window limiter keyed by client IP, used to slow
// down brute-force login attempts.
type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{hits: map[string][]time.Time{}, limit: limit, window: window}
}

// allow reports whether key may proceed, recording the attempt.
func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-r.window)
	kept := r.hits[key][:0]
	for _, t := range r.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= r.limit {
		r.hits[key] = kept
		return false
	}
	r.hits[key] = append(kept, now)
	return true
}
