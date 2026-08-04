// Package middleware contains HTTP middleware shared by API modules.
package middleware

import (
	"net/http"
	"strings"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/L-ka080/DDB-6-case/internal/platform/security"
	"github.com/gin-gonic/gin"
)

const (
	// UserIDContextKey stores the authenticated user ID in a Gin context.
	UserIDContextKey = "user_id"
	// UserLoginContextKey stores the authenticated login in a Gin context.
	UserLoginContextKey = "user_login"
)

// TokenParser is the JWT verification capability required by the middleware.
type TokenParser interface {
	Parse(rawToken string) (security.Claims, error)
}

// Auth verifies a Bearer access token and stores its identity in the request context.
func Auth(parser TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeUnauthorized(c)
			return
		}

		claims, err := parser.Parse(parts[1])
		if err != nil {
			writeUnauthorized(c)
			return
		}

		c.Set(UserIDContextKey, claims.UserID)
		c.Set(UserLoginContextKey, claims.Login)
		c.Next()
	}
}

func writeUnauthorized(c *gin.Context) {
	httpserver.WriteError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
}
