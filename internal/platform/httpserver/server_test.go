package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/L-ka080/DDB-6-case/docs"
	"github.com/gin-gonic/gin"
)

type pingFunc func(context.Context) error

func (fn pingFunc) Ping(ctx context.Context) error {
	return fn(ctx)
}

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := New("8080", availableDependencies(), t.TempDir(), testRoutes())

	response := performRequest(server, http.MethodGet, "/health", "")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(response.Body.String()); got != `{"status":"ok"}` {
		t.Fatalf("body = %s, want health response", got)
	}
}

func TestReady(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		databaseError error
		redisError    error
		wantStatus    int
		wantBody      string
	}{
		{
			name:       "database available",
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ready"}`,
		},
		{
			name:          "database unavailable",
			databaseError: errors.New("connection refused"),
			wantStatus:    http.StatusServiceUnavailable,
			wantBody:      `{"error":{"code":"database_unavailable","message":"database is unavailable"}}`,
		},
		{
			name:       "Redis unavailable",
			redisError: errors.New("connection refused"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"error":{"code":"redis_unavailable","message":"redis is unavailable"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := New("8080", ReadinessDependencies{
				Database: pingFunc(func(context.Context) error { return tt.databaseError }),
				Redis:    pingFunc(func(context.Context) error { return tt.redisError }),
			}, t.TempDir(), testRoutes())
			response := performRequest(server, http.MethodGet, "/ready", "")

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(response.Body.String()); got != tt.wantBody {
				t.Fatalf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := New("8080", availableDependencies(), t.TempDir(), testRoutes())

	response := performRequest(server, http.MethodOptions, "/health", "http://localhost:5173")

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestSwagger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := New("8080", availableDependencies(), t.TempDir(), testRoutes())

	tests := []struct {
		path        string
		contentType string
		bodyPart    string
	}{
		{
			path:        "/swagger/index.html",
			contentType: "text/html",
			bodyPart:    "Swagger UI",
		},
		{
			path:        "/swagger/doc.json",
			contentType: "application/json",
			bodyPart:    "Exchange Chain API",
		},
		{
			path:        "/swagger/swagger-ui.css",
			contentType: "text/css",
			bodyPart:    ".swagger-ui",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			response := performRequest(server, http.MethodGet, tt.path, "")

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if got := response.Header().Get("Content-Type"); !strings.Contains(got, tt.contentType) {
				t.Fatalf("Content-Type = %q, want it to contain %q", got, tt.contentType)
			}
			if got := response.Body.String(); !strings.Contains(got, tt.bodyPart) {
				t.Fatalf("body does not contain %q", tt.bodyPart)
			}
		})
	}
}

func TestHTTPErrorFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := New("8080", availableDependencies(), t.TempDir(), testRoutes())

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":{"code":"not_found","message":"resource not found"}}`,
		},
		{
			name:       "unsupported method",
			method:     http.MethodPost,
			path:       "/health",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `{"error":{"code":"method_not_allowed","message":"method not allowed"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performRequest(server, tt.method, tt.path, "")

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(response.Body.String()); got != tt.wantBody {
				t.Fatalf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}

func testRoutes() Routes {
	noop := func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	}
	pass := func(c *gin.Context) {
		c.Next()
	}

	return Routes{
		Register:       noop,
		Login:          noop,
		Me:             noop,
		CreateItem:     noop,
		CreateChain:    noop,
		DeclineChain:   noop,
		AuthMiddleware: pass,
	}
}

func availableDependencies() ReadinessDependencies {
	available := pingFunc(func(context.Context) error { return nil })
	return ReadinessDependencies{Database: available, Redis: available}
}

func performRequest(server *Server, method, path, origin string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	server.httpServer.Handler.ServeHTTP(recorder, request)

	return recorder
}
