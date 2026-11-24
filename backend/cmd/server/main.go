package main

import (
	"log"
	"net/http"
	"os"

	"b2b-crm/internal/db"
	"b2b-crm/internal/handlers"
	"b2b-crm/internal/middleware"
	"b2b-crm/internal/response"

	"github.com/gin-gonic/gin"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dbPath := getEnv("CRM_DB_PATH", "/opt/crm/data/crm.db")
	listenAddr := getEnv("CRM_LISTEN", "127.0.0.1:8080")
	jwtSecret := getEnv("CRM_JWT_SECRET", "REDACTED")

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	database, err := db.Init(dbPath)
	if err != nil {
		log.Fatalf("init db failed: %v", err)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Printf("panic recovered: %v", recovered)
		response.Error(c, http.StatusInternalServerError, "服务器内部异常")
		c.Abort()
	}))

	authHandler := &handlers.AuthHandler{DB: database, JWTSecret: jwtSecret}
	crmHandler := &handlers.CRMHandler{DB: database}

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			response.OK(c, gin.H{"status": "ok", "db": "sqlite-wal"})
		})
		v1.POST("/auth/login", authHandler.Login)

		auth := v1.Group("")
		auth.Use(middleware.AuthRequired(jwtSecret))
		{
			auth.GET("/auth/me", authHandler.Me)
			auth.POST("/auth/change-password", authHandler.ChangePassword)

			auth.GET("/dashboard", crmHandler.Dashboard)

			auth.GET("/companies", crmHandler.ListCompanies)
			auth.POST("/companies", crmHandler.CreateCompany)
			auth.PUT("/companies/:id", crmHandler.UpdateCompany)
			auth.DELETE("/companies/:id", crmHandler.DeleteCompany)
			auth.GET("/companies/:id/chain-summary", crmHandler.CompanyChainSummary)

			auth.GET("/contacts", crmHandler.ListContacts)
			auth.POST("/contacts", crmHandler.CreateContact)
			auth.PUT("/contacts/:id", crmHandler.UpdateContact)
			auth.DELETE("/contacts/:id", crmHandler.DeleteContact)

			auth.GET("/projects", crmHandler.ListProjects)
			auth.POST("/projects", crmHandler.CreateProject)
			auth.PUT("/projects/:id", crmHandler.UpdateProject)
			auth.DELETE("/projects/:id", crmHandler.DeleteProject)
			auth.POST("/projects/:id/contacts", crmHandler.UpsertProjectContact)
			auth.DELETE("/projects/:id/contacts/:contactId", crmHandler.RemoveProjectContact)

			auth.GET("/follow-ups", crmHandler.ListFollowUps)
			auth.POST("/follow-ups", crmHandler.CreateFollowUp)
			auth.PUT("/follow-ups/:id", crmHandler.UpdateFollowUp)
			auth.DELETE("/follow-ups/:id", crmHandler.DeleteFollowUp)
			auth.GET("/calendar/follow-ups", crmHandler.CalendarFollowUps)
		}
	}

	log.Printf("b2b-crm server listening on %s (db=%s)", listenAddr, dbPath)
	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
