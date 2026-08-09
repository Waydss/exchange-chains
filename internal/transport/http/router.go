package transporthttp

import (
	"context"
	"log"
	stdhttp "net/http"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/gin-gonic/gin"
)

const readyCheckTimeout = 2 * time.Second

type Routes struct {
	Register   gin.HandlerFunc
	Login      gin.HandlerFunc
	Me         gin.HandlerFunc
	CreateItem gin.HandlerFunc
	ListItems  gin.HandlerFunc
	GetItem    gin.HandlerFunc
	ListWishes gin.HandlerFunc

	// FIXME Remove this at one point...
	CreateChain gin.HandlerFunc

	GetChainsByUser gin.HandlerFunc
	GetChainsByItem gin.HandlerFunc

	UpdatePublicRating gin.HandlerFunc
	GetUserRatings     gin.HandlerFunc

	GetChainByID   gin.HandlerFunc
	AcceptChain    gin.HandlerFunc
	DeclineChain   gin.HandlerFunc
	AuthMiddleware gin.HandlerFunc
}

func NewRouter(
	dependencies httpserver.ReadinessDependencies,
	uploadDirectory string,
	routes Routes,
) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(
		gin.Logger(),
		gin.CustomRecovery(func(c *gin.Context, recovered any) {
			log.Printf("recovered from HTTP handler panic: %v", recovered)
			httpserver.WriteError(c, stdhttp.StatusInternalServerError, "internal_error", "internal server error")
		}),
		cors(),
	)

	router.GET("/health", healthHandler)
	router.GET("/ready", readyHandler(dependencies))
	router.GET("/swagger/*any", httpserver.SwaggerHandler())
	router.Static("/uploads", uploadDirectory)

	api := router.Group("/api/v1")
	auth := api.Group("/auth")
	auth.POST("/register", routes.Register)
	auth.POST("/login", routes.Login)

	protected := api.Group("")
	protected.Use(routes.AuthMiddleware)
	protected.GET("/me", routes.Me)
	protected.POST("/items", routes.CreateItem)
	protected.GET("/items", routes.ListItems)
	protected.GET("/items/:itemId", routes.GetItem)
	protected.GET("/wishes", routes.ListWishes)
	protected.POST("/chain", routes.CreateChain)
	protected.GET("/chains/by-user", routes.GetChainsByUser)
	protected.GET("/chains/by-item", routes.GetChainsByItem)
	protected.GET("/chains/:chainId", routes.GetChainByID)
	protected.POST("/chains/:chainId/accept", routes.AcceptChain)
	protected.POST("/chains/:chainId/decline", routes.DeclineChain)
	protected.POST("/ratings", routes.UpdatePublicRating)
	protected.GET("/ratings", routes.GetUserRatings)

	router.NoRoute(func(c *gin.Context) {
		httpserver.WriteError(c, stdhttp.StatusNotFound, "not_found", "resource not found")
	})
	router.NoMethod(func(c *gin.Context) {
		httpserver.WriteError(c, stdhttp.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	return router
}

func healthHandler(c *gin.Context) {
	c.JSON(stdhttp.StatusOK, httpserver.HealthResponse{Status: "ok"})
}

func readyHandler(dependencies httpserver.ReadinessDependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), readyCheckTimeout)
		defer cancel()

		if err := dependencies.Database.Ping(ctx); err != nil {
			log.Printf("readiness check failed: %v", err)
			httpserver.WriteError(c, stdhttp.StatusServiceUnavailable, "database_unavailable", "database is unavailable")
			return
		}
		if err := dependencies.Redis.Ping(ctx); err != nil {
			log.Printf("Redis readiness check failed: %v", err)
			httpserver.WriteError(c, stdhttp.StatusServiceUnavailable, "redis_unavailable", "redis is unavailable")
			return
		}

		c.JSON(stdhttp.StatusOK, httpserver.ReadinessResponse{Status: "ready"})
	}
}

func cors() gin.HandlerFunc {
	allowedOrigins := map[string]struct{}{
		"http://localhost:3000": {},
		"http://localhost:5173": {},
		"http://127.0.0.1:3000": {},
		"http://127.0.0.1:5173": {},
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowedOrigins[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == stdhttp.MethodOptions {
			c.AbortWithStatus(stdhttp.StatusNoContent)
			return
		}

		c.Next()
	}
}
