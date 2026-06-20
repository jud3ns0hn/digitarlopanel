package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/config"
)

// handleSettingsGet returns the panel configuration without secrets.
func (s *Server) handleSettingsGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"listen":               s.cfg.Listen,
		"data_dir":             s.cfg.DataDir,
		"file_root":            s.cfg.FileRoot,
		"backup_dir":           s.cfg.BackupDir,
		"ftp_user":             s.cfg.FTPUser,
		"ftp_group":            s.cfg.FTPGroup,
		"tls_enabled":          s.cfg.TLSEnabled,
		"tls_auto_self_signed": s.cfg.TLSAutoSelfSigned,
		"ai_provider":          s.cfg.AIProvider,
		"ai_base_url":          s.cfg.AIBaseURL,
		"ai_model":             s.cfg.AIModel,
		"ai_key_set":           s.cfg.AIAPIKey != "",
		"mcp_token":            s.cfg.MCPToken,
		"os":                   s.os,
	})
}

type settingsUpdateRequest struct {
	FileRoot          *string `json:"file_root"`
	BackupDir         *string `json:"backup_dir"`
	FTPUser           *string `json:"ftp_user"`
	FTPGroup          *string `json:"ftp_group"`
	Listen            *string `json:"listen"`
	TLSEnabled        *bool   `json:"tls_enabled"`
	TLSAutoSelfSigned *bool   `json:"tls_auto_self_signed"`
	AIProvider        *string `json:"ai_provider"`
	AIBaseURL         *string `json:"ai_base_url"`
	AIAPIKey          *string `json:"ai_api_key"`
	AIModel           *string `json:"ai_model"`
}

// handleSettingsUpdate persists configuration changes. Changes to listen and TLS
// take effect on the next restart; file_root/backup_dir apply immediately.
func (s *Server) handleSettingsUpdate(c *gin.Context) {
	var req settingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid settings payload")
		return
	}
	if req.FileRoot != nil {
		s.cfg.FileRoot = *req.FileRoot
	}
	if req.BackupDir != nil {
		s.cfg.BackupDir = *req.BackupDir
	}
	if req.FTPUser != nil {
		s.cfg.FTPUser = *req.FTPUser
	}
	if req.FTPGroup != nil {
		s.cfg.FTPGroup = *req.FTPGroup
	}
	if req.Listen != nil {
		s.cfg.Listen = *req.Listen
	}
	if req.TLSEnabled != nil {
		s.cfg.TLSEnabled = *req.TLSEnabled
	}
	if req.TLSAutoSelfSigned != nil {
		s.cfg.TLSAutoSelfSigned = *req.TLSAutoSelfSigned
	}
	if req.AIProvider != nil {
		s.cfg.AIProvider = *req.AIProvider
	}
	if req.AIBaseURL != nil {
		s.cfg.AIBaseURL = *req.AIBaseURL
	}
	if req.AIAPIKey != nil && *req.AIAPIKey != "" {
		s.cfg.AIAPIKey = *req.AIAPIKey
	}
	if req.AIModel != nil {
		s.cfg.AIModel = *req.AIModel
	}

	if s.cfgPath != "" {
		if err := config.Save(s.cfgPath, s.cfg); err != nil {
			serverError(c, err)
			return
		}
	}
	s.audit(c, "settings_update", "panel configuration changed")
	c.JSON(http.StatusOK, gin.H{"status": "ok", "restart_required": req.Listen != nil || req.TLSEnabled != nil})
}
