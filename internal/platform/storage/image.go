package storage

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/L-ka080/DDB-6-case/internal/item"
)

const (
	mimeProbeSize      = 512
	randomFilenameSize = 16
)

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func prepareImageUpload(upload item.PhotoUpload, maxFileSize int64) (*bufio.Reader, string, string, error) {
	if upload.Size > maxFileSize {
		return nil, "", "", item.ErrImageTooLarge
	}

	reader := bufio.NewReader(upload.Reader)
	header, err := reader.Peek(mimeProbeSize)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return nil, "", "", fmt.Errorf("%w: inspect uploaded image", item.ErrFileStorage)
	}

	contentType := http.DetectContentType(header)
	extension, supported := allowedImageTypes[contentType]
	if !supported {
		return nil, "", "", item.ErrUnsupportedImageType
	}

	return reader, contentType, extension, nil
}
