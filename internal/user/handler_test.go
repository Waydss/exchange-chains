package user

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
)

type handlerServiceStub struct {
	registerResult AuthResult
	registerError  error
	loginResult    AuthResult
	loginError     error
	user           PublicUser
	getByIDError   error

	updatePublicRatingResult TotalUserRatingDTO
	updatePublicRatingError  error
	getUserRatingsResult     UserRatingsDTO
	getUserRatingsError      error
}

func (s handlerServiceStub) Register(context.Context, RegisterInput) (AuthResult, error) {
	return s.registerResult, s.registerError
}

func (s handlerServiceStub) Login(context.Context, LoginInput) (AuthResult, error) {
	return s.loginResult, s.loginError
}

func (s handlerServiceStub) GetByID(context.Context, int64) (PublicUser, error) {
	return s.user, s.getByIDError
}

func (s handlerServiceStub) UpdatePublicRating(context.Context, int64, int64, float64, int64) (TotalUserRatingDTO, error) {
	return s.updatePublicRatingResult, s.updatePublicRatingError
}

func (s handlerServiceStub) GetUserRatings(context.Context, int64) (UserRatingsDTO, error) {
	return s.getUserRatingsResult, s.getUserRatingsError
}

func TestRegisterHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authResult := testAuthResult()

	tests := []struct {
		name       string
		body       string
		service    handlerServiceStub
		wantStatus int
		wantCode   string
	}{
		{
			name:       "created",
			body:       `{"login":"vadim","password":"password123","name":"Вадим"}`,
			service:    handlerServiceStub{registerResult: authResult},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "occupied login",
			body:       `{"login":"vadim","password":"password123","name":"Вадим"}`,
			service:    handlerServiceStub{registerError: ErrLoginAlreadyExists},
			wantStatus: http.StatusConflict,
			wantCode:   "login_already_exists",
		},
		{
			name:       "invalid JSON",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation_error",
		},
		{
			name:       "empty body",
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.service)
			response := performHandlerRequest(http.MethodPost, "/register", tt.body, handler.Register, 0)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if tt.wantCode != "" && !strings.Contains(response.Body.String(), `"code":"`+tt.wantCode+`"`) {
				t.Fatalf("body = %s, want error code %q", response.Body.String(), tt.wantCode)
			}
			assertNoPrivateFields(t, response.Body.String())
			if tt.wantStatus == http.StatusCreated && !strings.Contains(response.Body.String(), `"name":"Вадим"`) {
				t.Fatalf("body = %s, want registered user name", response.Body.String())
			}
		})
	}
}

func TestLoginHandlerHidesCredentialReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(handlerServiceStub{loginError: ErrInvalidCredentials})

	response := performHandlerRequest(
		http.MethodPost,
		"/login",
		`{"login":"unknown","password":"wrong-password"}`,
		handler.Login,
		0,
	)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(response.Body.String(), `"code":"invalid_credentials"`) {
		t.Fatalf("body = %s, want invalid_credentials", response.Body.String())
	}
}

func TestLoginHandlerReturnsAuthResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(handlerServiceStub{loginResult: testAuthResult()})

	response := performHandlerRequest(
		http.MethodPost,
		"/login",
		`{"login":"vadim","password":"password123"}`,
		handler.Login,
		0,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"token_type":"Bearer"`) {
		t.Fatalf("body = %s, want Bearer auth result", response.Body.String())
	}
	assertNoPrivateFields(t, response.Body.String())
}

func TestMeHandlerReturnsPublicUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	publicUser := testAuthResult().User
	handler := NewHandler(handlerServiceStub{user: publicUser})

	response := performHandlerRequest(http.MethodGet, "/me", "", handler.Me, 7)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"login":"vadim"`) {
		t.Fatalf("body = %s, want public user", response.Body.String())
	}
	assertNoPrivateFields(t, response.Body.String())
}

func TestMeHandlerReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(handlerServiceStub{getByIDError: fmt.Errorf("load user: %w", ErrUserNotFound)})

	response := performHandlerRequest(http.MethodGet, "/me", "", handler.Me, 7)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func performHandlerRequest(
	method string,
	path string,
	body string,
	handler gin.HandlerFunc,
	userID int64,
) *httptest.ResponseRecorder {
	router := gin.New()
	router.Handle(method, path, func(c *gin.Context) {
		if userID > 0 {
			c.Set(middleware.UserIDContextKey, userID)
		}
		handler(c)
	})

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

func testAuthResult() AuthResult {
	return AuthResult{
		User: PublicUser{
			ID:           7,
			Login:        "vadim",
			Name:         "Вадим",
			PublicRating: 0,
			CreatedAt:    time.Date(2026, time.August, 3, 17, 0, 0, 0, time.UTC),
		},
		AccessToken: "access-token",
		TokenType:   "Bearer",
		ExpiresIn:   86400,
	}
}

func assertNoPrivateFields(t *testing.T, body string) {
	t.Helper()
	if strings.Contains(body, "password") || strings.Contains(body, "shadow_rating") {
		t.Fatalf("response contains a private field: %s", body)
	}
}
