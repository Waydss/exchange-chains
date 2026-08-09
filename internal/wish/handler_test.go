package wish

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
)

type handlerServiceStub struct {
	userID int64
}

func (s *handlerServiceStub) ListByUserID(_ context.Context, userID int64) ([]ReadResponse, error) {
	s.userID = userID
	return []ReadResponse{}, nil
}

func TestListUsesAuthenticatedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerServiceStub{}
	handler := NewHandler(service)
	router := gin.New()
	router.GET("/wishes", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.List(c)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/wishes", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if service.userID != 42 {
		t.Fatalf("user ID = %d, want 42", service.userID)
	}
}
