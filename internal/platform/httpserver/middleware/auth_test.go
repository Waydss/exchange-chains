package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/L-ka080/DDB-6-case/internal/platform/security"
	"github.com/gin-gonic/gin"
)

type parseFunc func(string) (security.Claims, error)

func (fn parseFunc) Parse(rawToken string) (security.Claims, error) {
	return fn(rawToken)
}

func TestAuthAcceptsValidBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", Auth(parseFunc(func(token string) (security.Claims, error) {
		if token != "valid-token" {
			return security.Claims{}, errors.New("unexpected token")
		}
		return security.Claims{UserID: 42, Login: "vadim"}, nil
	})), func(c *gin.Context) {
		userID, _ := c.Get(UserIDContextKey)
		login, _ := c.Get(UserLoginContextKey)
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "login": login})
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body struct {
		UserID int64  `json:"user_id"`
		Login  string `json:"login"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.UserID != 42 || body.Login != "vadim" {
		t.Fatalf("authenticated context = %+v, want user 42 with login vadim", body)
	}
}

func TestAuthRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", Auth(parseFunc(func(string) (security.Claims, error) {
		return security.Claims{}, errors.New("invalid token")
	})), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name          string
		authorization string
	}{
		{name: "missing header"},
		{name: "wrong scheme", authorization: "Basic token"},
		{name: "extra value", authorization: "Bearer token extra"},
		{name: "invalid JWT", authorization: "Bearer invalid-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", tt.authorization)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
		})
	}
}
