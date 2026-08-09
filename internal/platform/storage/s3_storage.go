package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/L-ka080/DDB-6-case/internal/item"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config contains settings for S3-compatible object storage.
type S3Config struct {
	Endpoint      string
	AccessKey     string
	SecretKey     string
	Bucket        string
	Region        string
	PublicBaseURL string
	UseSSL        bool
	MaxFileSize   int64
	EnsureBucket  bool
}

// S3Storage stores item images in an S3-compatible bucket.
type S3Storage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
	maxFileSize   int64
}

// NewS3Storage prepares an S3-compatible storage adapter.
func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	if cfg.EnsureBucket {
		exists, err := client.BucketExists(ctx, cfg.Bucket)
		if err != nil {
			return nil, fmt.Errorf("check S3 bucket: %w", err)
		}
		if !exists {
			if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
				return nil, fmt.Errorf("create S3 bucket: %w", err)
			}
		}
	}

	return &S3Storage{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
		maxFileSize:   cfg.MaxFileSize,
	}, nil
}

// SaveItemPhotos validates and uploads all photos for one item.
func (s *S3Storage) SaveItemPhotos(
	ctx context.Context,
	itemID int64,
	photos []item.PhotoUpload,
) ([]item.StoredPhoto, error) {
	stored := make([]item.StoredPhoto, 0, len(photos))
	if len(photos) == 0 {
		return stored, nil
	}

	for _, upload := range photos {
		if err := ctx.Err(); err != nil {
			_ = s.DeletePhotos(context.Background(), storedObjectKeys(stored))
			return nil, err
		}

		storedPhoto, err := s.savePhoto(ctx, itemID, upload)
		if err != nil {
			_ = s.DeletePhotos(context.Background(), storedObjectKeys(stored))
			return nil, err
		}
		stored = append(stored, storedPhoto)
	}

	return stored, nil
}

// DeleteItemPhotos removes all uploaded objects for an item.
func (s *S3Storage) DeleteItemPhotos(ctx context.Context, itemID int64) error {
	if itemID <= 0 {
		return nil
	}

	objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    itemObjectPrefix(itemID),
		Recursive: true,
	})
	errorCh := s.client.RemoveObjects(ctx, s.bucket, objectCh, minio.RemoveObjectsOptions{})
	for removeError := range errorCh {
		if removeError.Err != nil {
			return fmt.Errorf("%w: remove S3 object %q: %v", item.ErrFileStorage, removeError.ObjectName, removeError.Err)
		}
	}

	return nil
}

// DeletePhotos removes selected uploaded objects.
func (s *S3Storage) DeletePhotos(ctx context.Context, objectKeys []string) error {
	for _, objectKey := range objectKeys {
		objectKey = strings.TrimLeft(objectKey, "/")
		if objectKey == "" {
			continue
		}
		if err := s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
			return fmt.Errorf("%w: remove S3 object %q: %v", item.ErrFileStorage, objectKey, err)
		}
	}

	return nil
}

// PublicURL converts a stored object key into the public object URL.
func (s *S3Storage) PublicURL(objectKey string) string {
	if objectKey == "" {
		return ""
	}

	return s.publicBaseURL + "/" + strings.TrimLeft(objectKey, "/")
}

func (s *S3Storage) savePhoto(ctx context.Context, itemID int64, upload item.PhotoUpload) (item.StoredPhoto, error) {
	reader, contentType, extension, err := prepareImageUpload(upload, s.maxFileSize)
	if err != nil {
		return item.StoredPhoto{}, err
	}

	filename, err := randomFilename(extension)
	if err != nil {
		return item.StoredPhoto{}, fmt.Errorf("%w: generate image filename", item.ErrFileStorage)
	}

	objectKey := path.Join(itemObjectPrefix(itemID), filename)
	_, err = s.client.PutObject(
		ctx,
		s.bucket,
		objectKey,
		io.LimitReader(reader, s.maxFileSize+1),
		upload.Size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return item.StoredPhoto{}, fmt.Errorf("%w: upload S3 object", item.ErrFileStorage)
	}

	return item.StoredPhoto{ObjectKey: objectKey}, nil
}

func itemObjectPrefix(itemID int64) string {
	return path.Join("items", strconv.FormatInt(itemID, 10)) + "/"
}
