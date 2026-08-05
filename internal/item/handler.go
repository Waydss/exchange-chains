package item

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
)

const multipartMemory = 1 << 20

// HandlerService contains item use cases required by the HTTP handler.
type HandlerService interface {
	Create(ctx context.Context, input CreateInput) (CreateResponse, error)
}

// Handler exposes item creation over HTTP.
type Handler struct {
	service        HandlerService
	maxRequestSize int64
	maxFileSize    int64
	maxPhotos      int
}

// NewHandler creates an item HTTP handler with upload limits.
func NewHandler(service HandlerService, maxFileSize int64, maxPhotos int) *Handler {
	return &Handler{
		service:        service,
		maxRequestSize: maxFileSize*int64(maxPhotos) + multipartMemory,
		maxFileSize:    maxFileSize,
		maxPhotos:      maxPhotos,
	}
}

// Create godoc
//
// @Summary Создание предмета
// @Description Создаёт предмет текущего пользователя и сохраняет до пяти фотографий.
// @Tags items
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param title formData string true "Название предмета"
// @Param description formData string false "Описание предмета"
// @Param condition formData string true "Состояние" Enums(new,like_new,good,fair,poor)
// @Param category_name formData string true "Категория предмета"
// @Param price_min formData integer true "Минимальная цена в рублях"
// @Param price_max formData integer true "Максимальная цена в рублях"
// @Param wanted_category_name formData string true "Желаемая категория обмена"
// @Param photos formData []file false "Фотографии JPEG, PNG или WEBP" collectionFormat(multi)
// @Success 201 {object} CreateResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 404 {object} httpserver.ErrorResponse
// @Failure 413 {object} httpserver.ErrorResponse
// @Failure 415 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/items [post]
func (h *Handler) Create(c *gin.Context) {
	userIDValue, ok := c.Get(middleware.UserIDContextKey)
	userID, validUserID := userIDValue.(int64)
	if !ok || !validUserID || userID <= 0 {
		httpserver.WriteError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxRequestSize)
	if err := c.Request.ParseMultipartForm(multipartMemory); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			httpserver.WriteError(c, http.StatusRequestEntityTooLarge, "image_too_large", "multipart request is too large")
			return
		}
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", "request must be valid multipart/form-data")
		return
	}
	defer func() {
		_ = c.Request.MultipartForm.RemoveAll()
	}()

	priceMin, err := parsePrice(c.Request.MultipartForm.Value, "price_min")
	if err != nil {
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	priceMax, err := parsePrice(c.Request.MultipartForm.Value, "price_max")
	if err != nil {
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	photoHeaders := c.Request.MultipartForm.File["photos"]
	if len(photoHeaders) > h.maxPhotos {
		httpserver.WriteError(c, http.StatusRequestEntityTooLarge, "too_many_photos", "too many photos")
		return
	}
	for _, header := range photoHeaders {
		if header.Size > h.maxFileSize {
			httpserver.WriteError(c, http.StatusRequestEntityTooLarge, "image_too_large", "image is too large")
			return
		}
	}

	photoUploads, closePhotos, err := openPhotos(photoHeaders)
	if err != nil {
		httpserver.WriteError(c, http.StatusInternalServerError, "file_storage_error", "could not read uploaded photos")
		return
	}
	defer closePhotos()

	result, err := h.service.Create(c.Request.Context(), CreateInput{
		UserID:             userID,
		Title:              firstValue(c.Request.MultipartForm.Value, "title"),
		Description:        firstValue(c.Request.MultipartForm.Value, "description"),
		Condition:          firstValue(c.Request.MultipartForm.Value, "condition"),
		CategoryName:       firstValue(c.Request.MultipartForm.Value, "category_name"),
		PriceMin:           priceMin,
		PriceMax:           priceMax,
		WantedCategoryName: firstValue(c.Request.MultipartForm.Value, "wanted_category_name"),
		Photos:             photoUploads,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

func parsePrice(values map[string][]string, field string) (int64, error) {
	rawValue := firstValue(values, field)
	value, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", field)
	}

	return value, nil
}

func firstValue(values map[string][]string, field string) string {
	fieldValues := values[field]
	if len(fieldValues) == 0 {
		return ""
	}

	return fieldValues[0]
}

func openPhotos(headers []*multipart.FileHeader) ([]PhotoUpload, func(), error) {
	uploads := make([]PhotoUpload, 0, len(headers))
	openedFiles := make([]multipart.File, 0, len(headers))
	closeFiles := func() {
		for _, file := range openedFiles {
			_ = file.Close()
		}
	}

	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			closeFiles()
			return nil, func() {}, err
		}
		openedFiles = append(openedFiles, file)
		uploads = append(uploads, PhotoUpload{Reader: file, Size: header.Size})
	}

	return uploads, closeFiles, nil
}

func handleError(c *gin.Context, err error) {
	var validationError *ValidationError
	switch {
	case errors.As(err, &validationError):
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", validationError.Message)
	case errors.Is(err, ErrCategoryNotFound):
		httpserver.WriteError(c, http.StatusNotFound, "category_not_found", "category not found")
	case errors.Is(err, ErrWantedCategoryNotFound):
		httpserver.WriteError(c, http.StatusNotFound, "wanted_category_not_found", "wanted category not found")
	case errors.Is(err, ErrTooManyPhotos):
		httpserver.WriteError(c, http.StatusRequestEntityTooLarge, "too_many_photos", "too many photos")
	case errors.Is(err, ErrImageTooLarge):
		httpserver.WriteError(c, http.StatusRequestEntityTooLarge, "image_too_large", "image is too large")
	case errors.Is(err, ErrUnsupportedImageType):
		httpserver.WriteError(c, http.StatusUnsupportedMediaType, "unsupported_image_type", "only JPEG, PNG, and WEBP images are supported")
	case errors.Is(err, ErrFileStorage):
		log.Printf("item file storage failed: %v", err)
		httpserver.WriteError(c, http.StatusInternalServerError, "file_storage_error", "could not store item photos")
	default:
		log.Printf("item request failed: %v", err)
		httpserver.WriteError(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
