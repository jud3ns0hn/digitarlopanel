package api

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// redisKeyPattern keeps keys free of whitespace so they can't break the argv.
var redisKeyPattern = regexp.MustCompile(`^[^\s]{1,512}$`)

func (s *Server) redisCLI(ctx context.Context, args ...string) (string, error) {
	res, err := s.runner.Run(ctx, "redis-cli", args...)
	return strings.TrimSpace(res.CombinedOutput()), err
}

func (s *Server) handleRedisInfo(c *gin.Context) {
	info, err := s.redisCLI(c.Request.Context(), "info", "server")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": false, "error": err.Error()})
		return
	}
	size, _ := s.redisCLI(c.Request.Context(), "dbsize")
	c.JSON(http.StatusOK, gin.H{"available": true, "info": info, "dbsize": size})
}

func (s *Server) handleRedisKeys(c *gin.Context) {
	pattern := c.DefaultQuery("pattern", "*")
	if !redisKeyPattern.MatchString(pattern) {
		badRequest(c, "invalid pattern")
		return
	}
	// SCAN avoids blocking on large keyspaces; cap the returned set.
	out, err := s.redisCLI(c.Request.Context(), "--scan", "--pattern", pattern)
	if err != nil {
		serverError(c, err)
		return
	}
	keys := []string{}
	for _, k := range strings.Split(out, "\n") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
		if len(keys) >= 500 {
			break
		}
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func (s *Server) handleRedisGet(c *gin.Context) {
	key := c.Query("key")
	if !redisKeyPattern.MatchString(key) {
		badRequest(c, "invalid key")
		return
	}
	typ, _ := s.redisCLI(c.Request.Context(), "type", key)
	val, _ := s.redisCLI(c.Request.Context(), "get", key)
	ttl, _ := s.redisCLI(c.Request.Context(), "ttl", key)
	c.JSON(http.StatusOK, gin.H{"key": key, "type": typ, "value": val, "ttl": ttl})
}

type redisSetRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

func (s *Server) handleRedisSet(c *gin.Context) {
	var req redisSetRequest
	if err := c.ShouldBindJSON(&req); err != nil || !redisKeyPattern.MatchString(req.Key) {
		badRequest(c, "valid key required")
		return
	}
	if _, err := s.redisCLI(c.Request.Context(), "set", req.Key, req.Value); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "redis_set", req.Key)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleRedisDelete(c *gin.Context) {
	key := c.Query("key")
	if !redisKeyPattern.MatchString(key) {
		badRequest(c, "invalid key")
		return
	}
	if _, err := s.redisCLI(c.Request.Context(), "del", key); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "redis_del", key)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
