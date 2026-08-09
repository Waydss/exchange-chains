package storage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/L-ka080/DDB-6-case/internal/item"
)

func TestLocalStorageSavesMultipleDetectedImageTypes(t *testing.T) {
	baseDirectory := t.TempDir()
	storage, err := NewLocalStorage(baseDirectory, 1024)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	jpeg := append([]byte{0xff, 0xd8, 0xff, 0xe0}, bytes.Repeat([]byte{0}, 600)...)
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 600)...)
	stored, err := storage.SaveItemPhotos(context.Background(), 101, []item.PhotoUpload{
		{Reader: bytes.NewReader(jpeg), Size: int64(len(jpeg))},
		{Reader: bytes.NewReader(png), Size: int64(len(png))},
	})
	if err != nil {
		t.Fatalf("SaveItemPhotos() error = %v", err)
	}
	if len(stored) != 2 || !strings.HasSuffix(stored[0].ObjectKey, ".jpg") || !strings.HasSuffix(stored[1].ObjectKey, ".png") {
		t.Fatalf("stored photos = %#v", stored)
	}

	for _, photo := range stored {
		filename := filepath.Base(photo.ObjectKey)
		if _, err := os.Stat(filepath.Join(baseDirectory, "items", "101", filename)); err != nil {
			t.Fatalf("stored file %q: %v", filename, err)
		}
		if got := storage.PublicURL(photo.ObjectKey); !strings.HasPrefix(got, "/uploads/items/101/") {
			t.Fatalf("PublicURL() = %q, want uploads URL", got)
		}
	}
}

func TestLocalStorageRejectsUnsupportedImageAndCleansDirectory(t *testing.T) {
	baseDirectory := t.TempDir()
	storage, err := NewLocalStorage(baseDirectory, 1024)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	jpeg := append([]byte{0xff, 0xd8, 0xff, 0xe0}, bytes.Repeat([]byte{0}, 600)...)

	_, err = storage.SaveItemPhotos(context.Background(), 101, []item.PhotoUpload{
		{Reader: bytes.NewReader(jpeg), Size: int64(len(jpeg))},
		{Reader: bytes.NewReader([]byte("plain text")), Size: 10},
	})
	if !errors.Is(err, item.ErrUnsupportedImageType) {
		t.Fatalf("SaveItemPhotos() error = %v, want ErrUnsupportedImageType", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDirectory, "items", "101")); !os.IsNotExist(statErr) {
		t.Fatalf("partial item directory was not removed: %v", statErr)
	}
}

func TestLocalStorageRejectsOversizedImage(t *testing.T) {
	storage, err := NewLocalStorage(t.TempDir(), 10)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	_, err = storage.SaveItemPhotos(context.Background(), 101, []item.PhotoUpload{
		{Reader: bytes.NewReader(bytes.Repeat([]byte{0xff}, 11)), Size: 11},
	})
	if !errors.Is(err, item.ErrImageTooLarge) {
		t.Fatalf("SaveItemPhotos() error = %v, want ErrImageTooLarge", err)
	}
}
