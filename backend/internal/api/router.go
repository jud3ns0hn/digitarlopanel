package api

import "github.com/gin-gonic/gin"

// registerRoutes mounts all API endpoints under the given group.
func (s *Server) registerRoutes(api *gin.RouterGroup) {
	api.GET("/health", s.handleHealth)
	api.POST("/login", s.handleLogin)

	auth := api.Group("")
	auth.Use(s.authRequired())
	{
		auth.GET("/me", s.handleMe)
		auth.POST("/change-password", s.handleChangePassword)

		// Dashboard / monitoring
		auth.GET("/system/host", s.handleHostInfo)
		auth.GET("/system/metrics", s.handleMetrics)
		auth.GET("/system/metrics/stream", s.handleMetricsStream)

		// File manager
		auth.GET("/files/list", s.handleFileList)
		auth.GET("/files/read", s.handleFileRead)
		auth.POST("/files/write", s.handleFileWrite)
		auth.POST("/files/mkdir", s.handleFileMkdir)
		auth.POST("/files/rename", s.handleFileRename)
		auth.POST("/files/delete", s.handleFileDelete)
		auth.POST("/files/chmod", s.handleFileChmod)
		auth.GET("/files/download", s.handleFileDownload)
		auth.POST("/files/upload", s.handleFileUpload)

		// Software / services
		auth.GET("/software/list", s.handleSoftwareList)
		auth.POST("/software/install", s.handleSoftwareInstall)
		auth.POST("/software/uninstall", s.handleSoftwareUninstall)
		auth.POST("/software/service", s.handleServiceAction)

		// Websites
		auth.GET("/websites", s.handleWebsiteList)
		auth.POST("/websites", s.handleWebsiteCreate)
		auth.POST("/websites/:id/toggle", s.handleWebsiteToggle)
		auth.DELETE("/websites/:id", s.handleWebsiteDelete)

		// Databases
		auth.GET("/databases", s.handleDatabaseList)
		auth.POST("/databases", s.handleDatabaseCreate)
		auth.DELETE("/databases/:id", s.handleDatabaseDelete)

		// Cron jobs
		auth.GET("/cron", s.handleCronList)
		auth.POST("/cron", s.handleCronCreate)
		auth.POST("/cron/:id/toggle", s.handleCronToggle)
		auth.DELETE("/cron/:id", s.handleCronDelete)

		// Audit log
		auth.GET("/audit", s.handleAuditList)
	}
}
