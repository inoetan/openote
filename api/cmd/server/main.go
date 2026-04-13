package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/inoetan/openote/api/internal/audit"
	"github.com/inoetan/openote/api/internal/auth"
	"github.com/inoetan/openote/api/internal/config"
	"github.com/inoetan/openote/api/internal/db"
	"github.com/inoetan/openote/api/internal/execution"
	"github.com/inoetan/openote/api/internal/job"
	"github.com/inoetan/openote/api/internal/notification"
	"github.com/inoetan/openote/api/internal/project"
	internalredis "github.com/inoetan/openote/api/internal/redis"
	"github.com/inoetan/openote/api/internal/schedule"
	"github.com/inoetan/openote/api/internal/user"
)

func main() {
	cfg := config.Load()

	// ---- Infrastructure ----
	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DBDsn)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	redisClient, err := internalredis.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	// ---- Migrations ----
	if err := runMigrations(cfg.DBDsn, cfg.MigrationsPath); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	// ---- Repositories ----
	userRepo := user.NewRepository(pool)
	projectRepo := project.NewRepository(pool)
	nodeRepo := job.NewNodeRepository(pool)
	jobRepo := job.NewJobRepository(pool)
	scheduleRepo := schedule.NewRepository(pool)
	execRepo := execution.NewRepository(pool)
	auditRepo := audit.NewRepository(pool)
	notifRepo := notification.NewRepository(pool)

	// ---- Scheduler ----
	sched := schedule.NewScheduler(scheduleRepo, redisClient)
	schedCtx, schedCancel := context.WithCancel(ctx)
	defer schedCancel()
	if err := sched.Start(schedCtx); err != nil {
		log.Fatalf("start scheduler: %v", err)
	}
	defer sched.Stop()

	// ---- Handlers ----
	authHandler := auth.NewHandler(userRepo, redisClient, cfg.JWTSecret)
	userHandler := user.NewHandler(userRepo)
	projectHandler := project.NewHandler(projectRepo)
	jobHandler := job.NewHandler(nodeRepo, jobRepo)
	scheduleHandler := schedule.NewHandler(scheduleRepo, sched)
	execHandler := execution.NewHandler(execRepo, redisClient)
	auditHandler := audit.NewHandler(auditRepo)
	notifHandler := notification.NewHandler(notifRepo)

	// ---- Echo ----
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
	}))

	// ---- Routes ----
	api := e.Group("/api/v1")

	// Auth (public)
	authGroup := api.Group("/auth")
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)
	authGroup.POST("/logout", authHandler.Logout)

	// Protected routes
	jwtMW := auth.JWTMiddleware(cfg.JWTSecret)

	// Users
	users := api.Group("/users", jwtMW)
	users.GET("", userHandler.List)
	users.POST("", userHandler.Create)
	users.PUT("/:id", userHandler.Update)
	users.DELETE("/:id", userHandler.Delete)

	// Projects
	projects := api.Group("/projects", jwtMW)
	projects.GET("", projectHandler.List)
	projects.POST("", projectHandler.Create)
	projects.GET("/:id", projectHandler.Get)
	projects.PUT("/:id", projectHandler.Update)
	projects.DELETE("/:id", projectHandler.Delete)

	// Node definitions (under project)
	projects.GET("/:pid/nodes", jobHandler.ListNodes)
	projects.POST("/:pid/nodes", jobHandler.CreateNode)
	projects.GET("/:pid/nodes/:id", jobHandler.GetNode)
	projects.PUT("/:pid/nodes/:id", jobHandler.UpdateNode)
	projects.DELETE("/:pid/nodes/:id", jobHandler.DeleteNode)

	// Jobs (under project)
	projects.GET("/:pid/jobs", jobHandler.ListJobs)
	projects.POST("/:pid/jobs", jobHandler.CreateJob)
	projects.GET("/:pid/jobs/:id", jobHandler.GetJob)
	projects.PUT("/:pid/jobs/:id", jobHandler.UpdateJob)
	projects.DELETE("/:pid/jobs/:id", jobHandler.DeleteJob)

	// Schedule (under project job)
	projects.GET("/:pid/jobs/:id/schedule", scheduleHandler.GetSchedule)
	projects.PUT("/:pid/jobs/:id/schedule", scheduleHandler.UpsertSchedule)
	projects.DELETE("/:pid/jobs/:id/schedule", scheduleHandler.DeleteSchedule)

	// Execution
	projects.POST("/:pid/jobs/:id/execute", execHandler.Execute)
	projects.GET("/:pid/executions", execHandler.ListExecutions)
	projects.GET("/:pid/executions/:eid", execHandler.GetExecution)
	projects.POST("/:pid/executions/:eid/abort", execHandler.AbortExecution)

	// WebSocket log streaming (no JWT middleware for WS; token can be passed as query param)
	api.GET("/executions/:eid/logs/stream", execHandler.StreamLogs)

	// Audit
	projects.GET("/:pid/audit", auditHandler.ListAuditEvents)

	// Notification rules
	projects.GET("/:pid/jobs/:id/notifications", notifHandler.ListRules)
	projects.POST("/:pid/jobs/:id/notifications", notifHandler.CreateRule)
	projects.PUT("/:pid/jobs/:id/notifications/:rid", notifHandler.UpdateRule)
	projects.DELETE("/:pid/jobs/:id/notifications/:rid", notifHandler.DeleteRule)

	// ---- Graceful shutdown ----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Printf("starting server on %s", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("server stopped")
}

func runMigrations(dsn, migrationsPath string) error {
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
