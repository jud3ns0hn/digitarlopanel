package api

import "github.com/gin-gonic/gin"

// registerRoutes mounts all API endpoints under the given group.
//
// Authorization tiers:
//   - read group (any authenticated user, incl. viewer): GET/read endpoints
//   - write group (operator + admin): state-changing endpoints
//   - admin group (admin only): user management
func (s *Server) registerRoutes(api *gin.RouterGroup) {
	api.GET("/health", s.handleHealth)
	api.POST("/login", s.handleLogin)

	read := api.Group("")
	read.Use(s.authRequired())
	write := api.Group("")
	write.Use(s.authRequired(), s.writeRole())
	admin := api.Group("")
	admin.Use(s.authRequired(), s.adminOnly())

	// Account self-service (any authenticated user).
	read.GET("/me", s.handleMe)
	read.POST("/logout", s.handleLogout)
	read.POST("/change-password", s.handleChangePassword)
	read.POST("/2fa/setup", s.handle2FASetup)
	read.POST("/2fa/enable", s.handle2FAEnable)
	read.POST("/2fa/disable", s.handle2FADisable)

	// Dashboard / monitoring (read-only).
	read.GET("/system/host", s.handleHostInfo)
	read.GET("/system/metrics", s.handleMetrics)
	read.GET("/system/metrics/stream", s.handleMetricsStream)
	read.GET("/system/processes", s.handleProcesses)
	read.GET("/system/history", s.handleMetricsHistory)

	// File manager.
	read.GET("/files/list", s.handleFileList)
	read.GET("/files/read", s.handleFileRead)
	read.GET("/files/download", s.handleFileDownload)
	write.POST("/files/write", s.handleFileWrite)
	write.POST("/files/mkdir", s.handleFileMkdir)
	write.POST("/files/rename", s.handleFileRename)
	write.POST("/files/delete", s.handleFileDelete)
	write.POST("/files/chmod", s.handleFileChmod)
	write.POST("/files/upload", s.handleFileUpload)
	write.POST("/files/compress", s.handleFileCompress)
	write.POST("/files/extract", s.handleFileExtract)
	write.POST("/files/download-url", s.handleRemoteDownload)

	// Software / services catalog.
	read.GET("/software/list", s.handleSoftwareList)
	write.POST("/software/install", s.handleSoftwareInstall)
	write.POST("/software/uninstall", s.handleSoftwareUninstall)
	write.POST("/software/service", s.handleServiceAction)

	// systemd services.
	read.GET("/services", s.handleServiceList)
	write.POST("/services/control", s.handleServiceControl)

	// Websites.
	read.GET("/websites", s.handleWebsiteList)
	write.POST("/websites", s.handleWebsiteCreate)
	write.POST("/websites/:id/toggle", s.handleWebsiteToggle)
	write.POST("/websites/:id/php", s.handleWebsitePHP)
	write.POST("/websites/:id/proxy", s.handleWebsiteProxy)
	write.POST("/websites/:id/config", s.handleWebsiteConfig)
	write.DELETE("/websites/:id", s.handleWebsiteDelete)

	// PHP versions.
	read.GET("/php", s.handlePHPList)
	write.POST("/php/install", s.handlePHPInstall)

	// Databases (MySQL/MariaDB).
	read.GET("/databases", s.handleDatabaseList)
	write.POST("/databases", s.handleDatabaseCreate)
	write.DELETE("/databases/:id", s.handleDatabaseDelete)

	// PostgreSQL.
	read.GET("/postgres", s.handlePostgresList)
	write.POST("/postgres", s.handlePostgresCreate)
	write.DELETE("/postgres/:id", s.handlePostgresDelete)

	// Redis.
	read.GET("/redis/info", s.handleRedisInfo)
	read.GET("/redis/keys", s.handleRedisKeys)
	read.GET("/redis/get", s.handleRedisGet)
	write.POST("/redis/set", s.handleRedisSet)
	write.DELETE("/redis/key", s.handleRedisDelete)

	// Cron jobs.
	read.GET("/cron", s.handleCronList)
	write.POST("/cron", s.handleCronCreate)
	write.POST("/cron/:id/toggle", s.handleCronToggle)
	write.DELETE("/cron/:id", s.handleCronDelete)

	// Firewall.
	read.GET("/firewall", s.handleFirewallStatus)
	write.POST("/firewall/allow", s.handleFirewallAllow)
	write.POST("/firewall/deny", s.handleFirewallDeny)

	// SSL certificates.
	read.GET("/ssl", s.handleSSLList)
	write.POST("/ssl/issue", s.handleSSLIssue)
	write.POST("/ssl/self-signed", s.handleSSLSelfSigned)
	write.DELETE("/ssl/:id", s.handleSSLDelete)

	// Docker.
	read.GET("/docker", s.handleDockerStatus)
	read.GET("/docker/networks", s.handleDockerNetworks)
	read.GET("/docker/volumes", s.handleDockerVolumes)
	read.GET("/docker/logs", s.handleDockerLogs)
	read.GET("/docker/stats", s.handleDockerStats)
	write.POST("/docker/container", s.handleDockerContainerAction)
	write.POST("/docker/pull", s.handleDockerPull)
	write.POST("/docker/prune", s.handleDockerPrune)

	// App store (curated one-click docker apps).
	read.GET("/appstore", s.handleAppStoreList)
	write.POST("/appstore/install", s.handleAppStoreInstall)

	// Docker Compose app stacks.
	read.GET("/compose", s.handleComposeList)
	read.GET("/compose/:id/status", s.handleComposeStatus)
	write.POST("/compose", s.handleComposeDeploy)
	write.POST("/compose/:id/down", s.handleComposeDown)

	// FTP accounts.
	read.GET("/ftp", s.handleFTPList)
	write.POST("/ftp", s.handleFTPCreate)
	write.POST("/ftp/:id/password", s.handleFTPPassword)
	write.DELETE("/ftp/:id", s.handleFTPDelete)

	// DNS zones and records.
	read.GET("/dns", s.handleDNSZoneList)
	write.POST("/dns", s.handleDNSZoneCreate)
	write.DELETE("/dns/:id", s.handleDNSZoneDelete)
	write.POST("/dns/:id/records", s.handleDNSRecordCreate)
	write.DELETE("/dns/:id/records/:rid", s.handleDNSRecordDelete)

	// Mail domains and accounts.
	read.GET("/mail", s.handleMailDomainList)
	write.POST("/mail/domains", s.handleMailDomainCreate)
	write.DELETE("/mail/domains/:id", s.handleMailDomainDelete)
	write.POST("/mail/accounts", s.handleMailAccountCreate)
	write.DELETE("/mail/accounts/:id", s.handleMailAccountDelete)

	// Logs.
	read.GET("/logs/journal", s.handleLogJournal)
	read.GET("/logs/file", s.handleLogFile)

	// Backups.
	read.GET("/backups", s.handleBackupList)
	read.GET("/backups/:id/download", s.handleBackupDownload)
	write.POST("/backups", s.handleBackupCreate)
	write.DELETE("/backups/:id", s.handleBackupDelete)

	// Scheduled backups.
	read.GET("/schedules", s.handleScheduleList)
	write.POST("/schedules", s.handleScheduleCreate)
	write.POST("/schedules/:id/toggle", s.handleScheduleToggle)
	write.DELETE("/schedules/:id", s.handleScheduleDelete)

	// Panel settings (admin only).
	admin.GET("/settings", s.handleSettingsGet)
	admin.POST("/settings", s.handleSettingsUpdate)

	// AI assistant.
	read.GET("/ai/status", s.handleAIStatus)
	read.POST("/ai/chat", s.handleAIChat)

	// Audit log (read-only).
	read.GET("/audit", s.handleAuditList)

	// Web terminal (admin only; grants a privileged shell).
	admin.GET("/terminal", s.handleTerminal)

	// User management (admin only).
	admin.GET("/users", s.handleUserList)
	admin.POST("/users", s.handleUserCreate)
	admin.POST("/users/:id/role", s.handleUserRole)
	admin.POST("/users/:id/password", s.handleUserResetPassword)
	admin.DELETE("/users/:id", s.handleUserDelete)
}
