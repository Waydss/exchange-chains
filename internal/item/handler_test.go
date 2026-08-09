package item

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
)

type handlerServiceStub struct {
	result          CreateResponse
	addPhotosResult AddPhotosResult
	replaceResult   ReplacePhotoResult
	listResult      []ReadResponse
	getResult       ReadResponse
	err             error
	input           CreateInput
	addPhotosInput  AddPhotosInput
	replaceInput    ReplacePhotoInput
	deletedUserID   int64
	deletedItemID   int64
	deletedPhotoID  int64
	listedUserID    int64
	requestedID     int64
}

func (s *handlerServiceStub) Create(_ context.Context, input CreateInput) (CreateResponse, error) {
	s.input = input
	return s.result, s.err
}

func (s *handlerServiceStub) ListByUserID(_ context.Context, userID int64) ([]ReadResponse, error) {
	s.listedUserID = userID
	return s.listResult, s.err
}

func (s *handlerServiceStub) GetByID(_ context.Context, itemID int64) (ReadResponse, error) {
	s.requestedID = itemID
	return s.getResult, s.err
}

func (s *handlerServiceStub) AddPhotos(_ context.Context, input AddPhotosInput) (AddPhotosResult, error) {
	s.addPhotosInput = input
	return s.addPhotosResult, s.err
}

func (s *handlerServiceStub) ReplacePhoto(_ context.Context, input ReplacePhotoInput) (ReplacePhotoResult, error) {
	s.replaceInput = input
	return s.replaceResult, s.err
}

func (s *handlerServiceStub) DeletePhoto(_ context.Context, userID, itemID, photoID int64) error {
	s.deletedUserID = userID
	s.deletedItemID = itemID
	s.deletedPhotoID = photoID
	return s.err
}

func TestCreateHandlerUsesAuthenticatedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerServiceStub{result: CreateResponse{ID: 101, Photos: []PhotoResponse{}}}
	handler := NewHandler(service, 10<<20, 5)
	body, contentType := multipartBody(t, map[string]string{
		"title":                 "Гитара Yamaha",
		"description":           "Исправна",
		"condition":             "good",
		"category_name":         "Гитары",
		"price_min":             "40000",
		"price_max":             "50000",
		"wanted_category_name":  "Палатки",
		"wanted_title_keywords": "Палатка туристическая",
		"user_id":               "999",
	}, 0)

	router := gin.New()
	router.POST("/items", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.Create(c)
	})
	request := httptest.NewRequest(http.MethodPost, "/items", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if service.input.UserID != 42 {
		t.Fatalf("service user ID = %d, want JWT context user 42", service.input.UserID)
	}
}

func TestCreateHandlerRejectsMoreThanFivePhotos(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerServiceStub{}
	handler := NewHandler(service, 10<<20, 5)
	body, contentType := multipartBody(t, map[string]string{
		"title":                 "Гитара Yamaha",
		"condition":             "good",
		"category_name":         "Гитары",
		"price_min":             "40000",
		"price_max":             "50000",
		"wanted_category_name":  "Палатки",
		"wanted_title_keywords": "Палатка туристическая",
	}, 6)

	router := gin.New()
	router.POST("/items", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.Create(c)
	})
	request := httptest.NewRequest(http.MethodPost, "/items", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if service.input.UserID != 0 {
		t.Fatal("service must not be called for too many photos")
	}
}

func TestCreateHandlerRejectsMissingJWTContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(&handlerServiceStub{}, 10<<20, 5)
	router := gin.New()
	router.POST("/items", handler.Create)
	request := httptest.NewRequest(http.MethodPost, "/items", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestListHandlerUsesAuthenticatedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerServiceStub{listResult: []ReadResponse{}}
	handler := NewHandler(service, 10<<20, 5)
	router := gin.New()
	router.GET("/items", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.List(c)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/items", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if service.listedUserID != 42 {
		t.Fatalf("listed user ID = %d, want 42", service.listedUserID)
	}
}

func TestGetByIDHandlerUsesPathID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerServiceStub{getResult: ReadResponse{ID: 101}}
	handler := NewHandler(service, 10<<20, 5)
	router := gin.New()
	router.GET("/items/:itemId", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.GetByID(c)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/items/101", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if service.requestedID != 101 {
		t.Fatalf("requested item ID = %d, want 101", service.requestedID)
	}
}

func TestAddPhotosHandlerUsesAuthenticatedUserAndPathID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerServiceStub{addPhotosResult: AddPhotosResult{Photos: []PhotoResult{{ID: 501, URL: "/uploads/items/101/photo.png"}}}}
	handler := NewHandler(service, 10<<20, 5)
	body, contentType := multipartBody(t, nil, 2)
	router := gin.New()
	router.POST("/items/:itemId/photos", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.AddPhotos(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/items/101/photos", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if service.addPhotosInput.UserID != 42 || service.addPhotosInput.ItemID != 101 || len(service.addPhotosInput.Photos) != 2 {
		t.Fatalf("add photos input = %+v", service.addPhotosInput)
	}
}

func TestDeletePhotoHandlerUsesAuthenticatedUserAndPathIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerServiceStub{}
	handler := NewHandler(service, 10<<20, 5)
	router := gin.New()
	router.DELETE("/items/:itemId/photos/:photoId", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.DeletePhoto(c)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/items/101/photos/501", nil))

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNoContent, response.Body.String())
	}
	if service.deletedUserID != 42 || service.deletedItemID != 101 || service.deletedPhotoID != 501 {
		t.Fatalf("deleted user/item/photo = %d/%d/%d", service.deletedUserID, service.deletedItemID, service.deletedPhotoID)
	}
}

func TestReplacePhotoHandlerUsesAuthenticatedUserAndPathIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerServiceStub{replaceResult: ReplacePhotoResult{Photo: PhotoResult{ID: 501, URL: "/uploads/items/101/photo.png"}}}
	handler := NewHandler(service, 10<<20, 5)
	body, contentType := singlePhotoBody(t)
	router := gin.New()
	router.PUT("/items/:itemId/photos/:photoId", func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(42))
		handler.ReplacePhoto(c)
	})

	request := httptest.NewRequest(http.MethodPut, "/items/101/photos/501", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.replaceInput.UserID != 42 || service.replaceInput.ItemID != 101 || service.replaceInput.PhotoID != 501 {
		t.Fatalf("replace input = %+v", service.replaceInput)
	}
	if service.replaceInput.Photo.Reader == nil {
		t.Fatal("replacement photo was not passed to service")
	}
}

func multipartBody(t *testing.T, fields map[string]string, photoCount int) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("write multipart field: %v", err)
		}
	}
	for index := 0; index < photoCount; index++ {
		part, err := writer.CreateFormFile("photos", fmt.Sprintf("photo-%d.png", index))
		if err != nil {
			t.Fatalf("create multipart photo: %v", err)
		}
		if _, err := part.Write([]byte("\x89PNG\r\n\x1a\n")); err != nil {
			t.Fatalf("write multipart photo: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

func singlePhotoBody(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("photo", "replacement.png")
	if err != nil {
		t.Fatalf("create multipart photo: %v", err)
	}
	if _, err := part.Write([]byte("\x89PNG\r\n\x1a\n")); err != nil {
		t.Fatalf("write multipart photo: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}
