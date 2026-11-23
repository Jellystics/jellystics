package router

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/middleware"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	"github.com/Jellystics/Jellystics/internal/service/scheduler"
	"github.com/Jellystics/Jellystics/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

func New(svcs *service.Container, repos *repository.Container, hub *ws.Hub, db *gorm.DB, cfg *config.Config, sched *scheduler.Scheduler) *gin.Engine {
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// ── Handlers ──────────────────────────────────────────────────────────────
	frontAuthH := handler.NewFrontendAuthHandler(repos, cfg, svcs.Auth)
	configH    := handler.NewConfigApiHandler(repos, svcs)
	configH.SetScheduler(sched)
	sessionsH  := handler.NewSessionsApiHandler(repos)
	tasksH     := handler.NewTasksApiHandler(svcs, repos, db)
	proxyH     := handler.NewProxyApiHandler(repos)
	webhookH   := handler.NewWebhookHandler(svcs)
	syncH      := handler.NewSyncHandler(svcs)
	userH      := handler.NewUserHandler(repos)
	backupH    := handler.NewBackupHandler(repos, hub, svcs)
	logsH      := handler.NewLogsHandler(repos)
	adminH     := handler.NewAdminApiHandler(repos, db)

	// ── Socket.IO (public) ────────────────────────────────────────────────────
	// Must be registered before NoRoute SPA fallback.
	// Only clients presenting a valid session token receive events.
	hub.SetAuthorizer(func(token string) bool {
		_, err := svcs.Auth.Authenticate(context.Background(), token)
		return err == nil
	})
	r.Any("/socket.io/*path", func(c *gin.Context) {
		hub.ServeHTTP(c.Writer, c.Request)
	})

	// ── Prometheus metrics ────────────────────────────────────────────────────
	// Requires a valid Jellystics API key (Authorization: Bearer <key>).
	// Keys are managed via Settings → API Keys in the dashboard.
	reg := prometheus.NewRegistry()
	reg.MustRegister(handler.NewMetricsCollector(repos, db))
	metricsHandler := gin.WrapH(promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	r.GET("/metrics", metricsAuth(repos), metricsHandler)

	// ── Image proxy (public, no auth) ─────────────────────────────────────────
	// Image paths are public (used by <img> tags) and strictly whitelisted;
	// getSessions/getAdminUsers/getRecentlyAdded require a valid session.
	r.GET("/proxy/*path", proxyH.Proxy(middleware.Auth(svcs.Auth)))
	r.POST("/proxy/validateSettings", middleware.SetupOrAdmin(svcs.Auth), proxyH.ValidateSettings)

	// ── Auth routes (public) ─────────────────────────────────────────────────
	auth := r.Group("/auth")
	{
		auth.GET("/isConfigured", frontAuthH.IsConfigured)
		auth.POST("/configSetup", middleware.SetupOrAdmin(svcs.Auth), frontAuthH.ConfigSetup)
		auth.POST("/check-server", frontAuthH.CheckServer)
		auth.POST("/jellyfin-login", frontAuthH.JellyfinLogin)
	}
	r.POST("/auth/logout", middleware.Auth(svcs.Auth), frontAuthH.Logout)
	r.GET("/auth/me", middleware.Auth(svcs.Auth), frontAuthH.Me)

	// ── Authenticated routes ──────────────────────────────────────────────────
	authed := r.Group("", middleware.Auth(svcs.Auth))

	// Sessions
	authed.GET("/sessions/current", sessionsH.Current)

	// Webhooks – note: fixed paths (/event-status, /toggle-event/:eventType)
	// must be registered before the /:id wildcard.
	webhooks := authed.Group("/webhooks")
	{
		webhooks.GET("/", webhookH.List)
		webhooks.GET("/event-status", webhookH.EventStatus)
		webhooks.POST("/toggle-event/:eventType", webhookH.ToggleEvent)
		webhooks.GET("/:id", webhookH.Get)
		webhooks.POST("/", webhookH.Create)
		webhooks.PUT("/:id", webhookH.Update)
		webhooks.DELETE("/:id", webhookH.Delete)
		webhooks.POST("/:id/test", webhookH.Test)
		webhooks.POST("/:id/trigger-monthly", webhookH.TriggerMonthly)
	}

	// Sync
	sync := authed.Group("/sync")
	{
		sync.POST("/full", syncH.FullSync)
		sync.POST("/libraries", syncH.SyncLibraries)
		sync.POST("/users", syncH.SyncUsers)
		sync.GET("/status", syncH.Status)
		sync.POST("/importPlaybackBackup", tasksH.ImportBackup)
		sync.POST("/fetchItem", syncH.FetchItem)
	}

	// Backup (all routes require a session: backups contain the Jellyfin API key)
	backup := authed.Group("/backup")
	{
		backup.GET("/files", backupH.List)
		backup.GET("/files/:name", backupH.Download)
		backup.GET("/beginBackup", backupH.Create)
		backup.DELETE("/files/:name", backupH.Delete)
		backup.GET("/restore/:filename", backupH.Restore)
		backup.POST("/upload", backupH.Upload)
	}

	// Logs
	logs := authed.Group("/logs")
	{
		logs.GET("/getLogs", logsH.GetLogs)
	}

	// API
	api := authed.Group("/api")
	{
		// Config
		api.GET("/getconfig", configH.GetConfig)
		api.POST("/setconfig", configH.SetConfig)
		api.GET("/keys", configH.GetKeys)
		api.POST("/keys", configH.CreateKey)
		api.DELETE("/keys", configH.DeleteKey)
		// Config settings
		api.POST("/setExternalUrl", configH.SetExternalUrl)
		api.POST("/setPreferredAdmin", configH.SetPreferredAdmin)
		api.POST("/setRequireLogin", configH.SetRequireLogin)
		api.POST("/updateCredentials", configH.UpdateCredentials)
		api.GET("/TrackedLibraries", configH.TrackedLibraries)
		api.POST("/setExcludedLibraries", configH.SetExcludedLibraries)
		api.GET("/UntrackedUsers", configH.UntrackedUsers)
		api.POST("/setUntrackedUsers", configH.SetUntrackedUsers)
		api.GET("/getTaskSettings", configH.GetTaskSettings)
		api.POST("/setTaskSettings", configH.SetTaskSettings)
		api.GET("/isFirstRun", configH.IsFirstRun)
		api.GET("/getActivityMonitorSettings", configH.GetActivityMonitorSettings)
		api.POST("/setActivityMonitorSettings", configH.SetActivityMonitorSettings)
		api.GET("/CheckForUpdates", configH.CheckForUpdates)
		// Jellystics plugin sync
		if svcs.PluginSync != nil {
			api.GET("/plugin/status", middleware.AdminOnly(), handler.PluginStatus(svcs.PluginSync))
			api.POST("/plugin/push-history", middleware.AdminOnly(), handler.PluginPushHistory(svcs.PluginSync))
			api.GET("/plugin/push-history/status", middleware.AdminOnly(), handler.PluginPushStatus(svcs.PluginSync))
			api.POST("/plugin/resync", middleware.AdminOnly(), handler.PluginResync(svcs.PluginSync))
		}
		// Tasks
		api.GET("/getTasks", tasksH.List)
		api.POST("/runTask/:name", tasksH.Run)
		api.GET("/stopTask", configH.StopTask)
		// Purge
		api.DELETE("/item/purge", adminH.PurgeItem)
		api.DELETE("/library/purge", adminH.PurgeLibrary)
		api.DELETE("/libraryItems/purge", adminH.PurgeLibraryItems)
		// Activity
		api.POST("/deletePlaybackActivity", adminH.DeletePlaybackActivity)
		// Backup table config
		api.GET("/getBackupTables", adminH.GetBackupTables)
		api.POST("/setExcludedBackupTable", adminH.SetExcludedBackupTable)
	}

	handler.NewV2Handler(db).Register(authed)

	// Users
	users := authed.Group("/api/users")
	{
		users.GET("", userH.List)
		users.GET("/:id", userH.Get)
	}

	return r
}

// metricsAuth enforces bearer-token auth on /metrics using the app's API keys.
// The caller must send: Authorization: Bearer <api-key>
// If no API keys are configured, the endpoint is public (bootstrapping convenience).
func metricsAuth(repos *repository.Container) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cfg, err := repos.Config.Get(ctx)
		if err != nil {
			c.AbortWithStatus(503)
			return
		}

		var keys []map[string]string
		if len(cfg.ApiKeys) > 0 {
			_ = json.Unmarshal(cfg.ApiKeys, &keys)
		}

		auth := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatus(401)
			return
		}
		token := auth[len(prefix):]
		for _, k := range keys {
			if k["key"] == token {
				c.Next()
				return
			}
		}
		c.Header("WWW-Authenticate", "Bearer")
		c.AbortWithStatus(401)
	}
}
