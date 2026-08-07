package rejection

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
)

func TestDeclineChainUsesJWTUserAndRecordsReceivedItem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &rejectionWriterStub{}
	useCase := NewDeclineChainUseCase(store)
	handler := NewHandler(useCase)
	router := gin.New()
	router.POST("/chains/:chainId/decline", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.DeclineChain(c)
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/chains/77/decline",
		bytes.NewBufferString(`{"received_item_id":205,"user_id":999}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if store.userID != 42 || store.itemID != 205 {
		t.Fatalf("stored rejection = user %d, item %d; want user 42, item 205", store.userID, store.itemID)
	}
	if got := response.Body.String(); got != `{"chain_id":77,"received_item_id":205,"status":"rejection_recorded"}` {
		t.Fatalf("response body = %s", got)
	}
}

func TestDeclineChainValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		path          string
		body          string
		includeUserID bool
		wantStatus    int
	}{
		{name: "missing JWT user", path: "/chains/77/decline", body: `{"received_item_id":205}`, wantStatus: http.StatusUnauthorized},
		{name: "invalid chain ID", path: "/chains/invalid/decline", body: `{"received_item_id":205}`, includeUserID: true, wantStatus: http.StatusBadRequest},
		{name: "invalid JSON", path: "/chains/77/decline", body: `{`, includeUserID: true, wantStatus: http.StatusBadRequest},
		{name: "missing item ID", path: "/chains/77/decline", body: `{}`, includeUserID: true, wantStatus: http.StatusBadRequest},
		{name: "negative item ID", path: "/chains/77/decline", body: `{"received_item_id":-1}`, includeUserID: true, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(NewDeclineChainUseCase(&rejectionWriterStub{}))
			router := gin.New()
			router.POST("/chains/:chainId/decline", func(c *gin.Context) {
				if tt.includeUserID {
					c.Set(middleware.UserIDContextKey, int64(42))
				}
				handler.DeclineChain(c)
			})
			request := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}

func TestDeclineChainReturnsInternalErrorWhenStoreFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &rejectionWriterStub{err: errors.New("Redis unavailable")}
	handler := NewHandler(NewDeclineChainUseCase(store))
	router := gin.New()
	router.POST("/chains/:chainId/decline", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.DeclineChain(c)
	})
	request := httptest.NewRequest(http.MethodPost, "/chains/77/decline", bytes.NewBufferString(`{"received_item_id":205}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

type rejectionWriterStub struct {
	userID int64
	itemID int64
	err    error
}

func (s *rejectionWriterStub) Reject(_ context.Context, userID, itemID int64) error {
	s.userID = userID
	s.itemID = itemID
	return s.err
}
