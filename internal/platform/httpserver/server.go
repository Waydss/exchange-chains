// Package httpserver configures and runs the application HTTP server.
package httpserver

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
	readyCheckTimeout = 2 * time.Second
)

// Pinger is the external dependency capability required by the readiness endpoint.
type Pinger interface {
	Ping(ctx context.Context) error
}

// ReadinessDependencies contains external services required to serve requests.
type ReadinessDependencies struct {
	Database Pinger
	Redis    Pinger
}

// Routes contains the application handlers mounted by the HTTP server.
type Routes struct {
	Register       gin.HandlerFunc
	Login          gin.HandlerFunc
	Me             gin.HandlerFunc
	CreateItem     gin.HandlerFunc
	CreateChain    gin.HandlerFunc
	DeclineChain   gin.HandlerFunc
	AuthMiddleware gin.HandlerFunc
}

// Server owns the HTTP transport and its lifecycle.
type Server struct {
	httpServer *http.Server
}

// New configures the Gin router and HTTP server.
func New(port string, dependencies ReadinessDependencies, uploadDirectory string, routes Routes) *Server {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(
		gin.Logger(),
		gin.CustomRecovery(func(c *gin.Context, recovered any) {
			log.Printf("recovered from HTTP handler panic: %v", recovered)
			WriteError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}),
		cors(),
	)

	router.GET("/health", healthHandler)
	router.GET("/ready", readyHandler(dependencies))
	router.GET("/swagger/*any", swaggerHandler())
	router.Static("/uploads", uploadDirectory)

	api := router.Group("/api/v1")
	auth := api.Group("/auth")
	auth.POST("/register", routes.Register)
	auth.POST("/login", routes.Login)

	protected := api.Group("")
	protected.Use(routes.AuthMiddleware)
	protected.GET("/me", routes.Me)
	protected.POST("/items", routes.CreateItem)
	protected.POST("/chain", routes.CreateChain)
	protected.POST("/chains/:chainId/decline", routes.DeclineChain)

	router.NoRoute(func(c *gin.Context) {
		WriteError(c, http.StatusNotFound, "not_found", "resource not found")
	})
	router.NoMethod(func(c *gin.Context) {
		WriteError(c, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	return &Server{
		httpServer: &http.Server{
			Addr:              ":" + port,
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

// Run starts accepting HTTP requests and blocks until the server stops.
func (s *Server) Run() error {
	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// healthHandler godoc
//
// @Summary Проверка работоспособности
// @Description Возвращает состояние HTTP-процесса без проверки внешних зависимостей.
// @Tags system
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

// readyHandler godoc
//
// @Summary Проверка готовности
// @Description Проверяет доступность PostgreSQL и Redis и сообщает, готово ли приложение принимать запросы.
// @Tags system
// @Produce json
// @Success 200 {object} ReadinessResponse
// @Failure 503 {object} ErrorResponse
// @Router /ready [get]
func readyHandler(dependencies ReadinessDependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), readyCheckTimeout)
		defer cancel()

		if err := dependencies.Database.Ping(ctx); err != nil {
			log.Printf("readiness check failed: %v", err)
			WriteError(c, http.StatusServiceUnavailable, "database_unavailable", "database is unavailable")
			return
		}
		if err := dependencies.Redis.Ping(ctx); err != nil {
			log.Printf("Redis readiness check failed: %v", err)
			WriteError(c, http.StatusServiceUnavailable, "redis_unavailable", "redis is unavailable")
			return
		}

		c.JSON(http.StatusOK, ReadinessResponse{Status: "ready"})
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

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
