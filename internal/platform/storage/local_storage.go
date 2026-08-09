package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/L-ka080/DDB-6-case/internal/item"
)

// LocalStorage stores item images below a configured local directory.
type LocalStorage struct {
	baseDir     string
	maxFileSize int64
}

// NewLocalStorage prepares a writable local storage directory.
func NewLocalStorage(baseDir string, maxFileSize int64) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload directory: %w", err)
	}

	return &LocalStorage{baseDir: baseDir, maxFileSize: maxFileSize}, nil
}

// SaveItemPhotos validates and stores all uploaded photos for one item.
func (s *LocalStorage) SaveItemPhotos(
	ctx context.Context,
	itemID int64,
	photos []item.PhotoUpload,
) ([]item.StoredPhoto, error) {
	stored := make([]item.StoredPhoto, 0, len(photos))
	if len(photos) == 0 {
		return stored, nil
	}

	itemDirectory := s.itemDirectory(itemID)
	if err := os.MkdirAll(itemDirectory, 0o750); err != nil {
		return nil, fmt.Errorf("%w: create item upload directory", item.ErrFileStorage)
	}

	for _, upload := range photos {
		if err := ctx.Err(); err != nil {
			_ = os.RemoveAll(itemDirectory)
			return nil, err
		}

		storedPhoto, err := s.savePhoto(itemDirectory, itemID, upload)
		if err != nil {
			_ = os.RemoveAll(itemDirectory)
			return nil, err
		}
		stored = append(stored, storedPhoto)
	}

	return stored, nil
}

// DeleteItemPhotos removes all locally stored images for an item.
func (s *LocalStorage) DeleteItemPhotos(_ context.Context, itemID int64) error {
	if itemID <= 0 {
		return nil
	}
	if err := os.RemoveAll(s.itemDirectory(itemID)); err != nil {
		return fmt.Errorf("%w: remove item upload directory", item.ErrFileStorage)
	}

	return nil
}

// PublicURL converts a stored object key into the route served by the API.
func (s *LocalStorage) PublicURL(objectKey string) string {
	return path.Join("/uploads", objectKey)
}

func (s *LocalStorage) savePhoto(
	itemDirectory string,
	itemID int64,
	upload item.PhotoUpload,
) (item.StoredPhoto, error) {
	reader, _, extension, err := prepareImageUpload(upload, s.maxFileSize)
	if err != nil {
		return item.StoredPhoto{}, err
	}

	filename, err := randomFilename(extension)
	if err != nil {
		return item.StoredPhoto{}, fmt.Errorf("%w: generate image filename", item.ErrFileStorage)
	}
	filePath := filepath.Join(itemDirectory, filename)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return item.StoredPhoto{}, fmt.Errorf("%w: create uploaded image", item.ErrFileStorage)
	}

	written, copyErr := io.Copy(file, io.LimitReader(reader, s.maxFileSize+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(filePath)
		return item.StoredPhoto{}, fmt.Errorf("%w: write uploaded image", item.ErrFileStorage)
	}
	if written > s.maxFileSize {
		_ = os.Remove(filePath)
		return item.StoredPhoto{}, item.ErrImageTooLarge
	}

	return item.StoredPhoto{
		ObjectKey: path.Join("items", strconv.FormatInt(itemID, 10), filename),
	}, nil
}

func (s *LocalStorage) itemDirectory(itemID int64) string {
	return filepath.Join(s.baseDir, "items", strconv.FormatInt(itemID, 10))
}

func randomFilename(extension string) (string, error) {
	randomBytes := make([]byte, randomFilenameSize)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes) + extension, nil
}
