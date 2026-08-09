// Package config loads and validates application configuration.
package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config contains validated runtime settings.
type Config struct {
	App       AppConfig       `envconfig:"APP"`
	HTTP      HTTPConfig      `envconfig:"HTTP"`
	PG        PGConfig        `envconfig:"POSTGRES"`
	Redis     RedisConfig     `envconfig:"REDIS"`
	JWT       JWTConfig       `envconfig:"JWT"`
	Upload    UploadConfig    `envconfig:"UPLOAD"`
	Storage   StorageConfig   `envconfig:"STORAGE"`
	S3        S3Config        `envconfig:"S3"`
	Rejection RejectionConfig `envconfig:"REJECTION"`
}

// AppConfig contains general application settings.
type AppConfig struct {
	Environment string `envconfig:"APP_ENV" default:"local"`
}

// HTTPConfig contains HTTP server settings.
type HTTPConfig struct {
	Port            string        `envconfig:"APP_PORT" default:"8080"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`
}

// PGConfig contains PostgreSQL settings.
type PGConfig struct {
	URL string `envconfig:"DATABASE_URL" required:"true"`
}

// RedisConfig contains Redis connection settings.
type RedisConfig struct {
	Addr         string        `envconfig:"REDIS_ADDR" default:"redis:6379"`
	Password     string        `envconfig:"REDIS_PASSWORD"`
	DB           int           `envconfig:"REDIS_DB" default:"0"`
	DialTimeout  time.Duration `envconfig:"REDIS_DIAL_TIMEOUT" default:"5s"`
	ReadTimeout  time.Duration `envconfig:"REDIS_READ_TIMEOUT" default:"2s"`
	WriteTimeout time.Duration `envconfig:"REDIS_WRITE_TIMEOUT" default:"2s"`
}

// JWTConfig contains access-token settings.
type JWTConfig struct {
	Secret string        `envconfig:"JWT_SECRET" required:"true"`
	TTL    time.Duration `envconfig:"JWT_TTL" default:"168h"`
}

// UploadConfig contains item photo upload settings.
type UploadConfig struct {
	Dir           string `envconfig:"UPLOAD_DIR" default:"/app/uploads"`
	MaxFileSize   int64  `envconfig:"MAX_UPLOAD_SIZE" default:"10485760"`
	MaxItemPhotos int    `envconfig:"MAX_ITEM_PHOTOS" default:"5"`
}

// StorageConfig selects the photo storage implementation.
type StorageConfig struct {
	Driver string `envconfig:"STORAGE_DRIVER" default:"local"`
}

// S3Config contains S3-compatible object storage settings.
type S3Config struct {
	Endpoint      string `envconfig:"S3_ENDPOINT" default:"minio:9000"`
	PublicBaseURL string `envconfig:"S3_PUBLIC_URL" default:"http://localhost:9000/item-photos"`
	Bucket        string `envconfig:"S3_BUCKET" default:"item-photos"`
	AccessKey     string `envconfig:"S3_ACCESS_KEY" default:"minioadmin"`
	SecretKey     string `envconfig:"S3_SECRET_KEY" default:"minioadmin"`
	Region        string `envconfig:"S3_REGION" default:"us-east-1"`
	UseSSL        bool   `envconfig:"S3_USE_SSL" default:"false"`
	EnsureBucket  bool   `envconfig:"S3_ENSURE_BUCKET" default:"true"`
}

// RejectionConfig contains temporary item rejection settings.
type RejectionConfig struct {
	ItemTTL time.Duration `envconfig:"REJECTED_ITEM_TTL" default:"336h"`
}

// Load reads and validates the application configuration.
func Load() (*Config, error) {
	return load(true)
}

func load(readDotEnv bool) (*Config, error) {
	if readDotEnv {
		_ = godotenv.Load()
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("process environment configuration: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	c.App.Environment = strings.TrimSpace(c.App.Environment)
	c.HTTP.Port = strings.TrimSpace(c.HTTP.Port)
	c.PG.URL = strings.TrimSpace(c.PG.URL)
	c.Redis.Addr = strings.TrimSpace(c.Redis.Addr)
	c.JWT.Secret = strings.TrimSpace(c.JWT.Secret)
	c.Upload.Dir = strings.TrimSpace(c.Upload.Dir)
	c.Storage.Driver = strings.ToLower(strings.TrimSpace(c.Storage.Driver))
	c.S3.Endpoint = strings.TrimSpace(c.S3.Endpoint)
	c.S3.PublicBaseURL = strings.TrimRight(strings.TrimSpace(c.S3.PublicBaseURL), "/")
	c.S3.Bucket = strings.TrimSpace(c.S3.Bucket)
	c.S3.AccessKey = strings.TrimSpace(c.S3.AccessKey)
	c.S3.SecretKey = strings.TrimSpace(c.S3.SecretKey)
	c.S3.Region = strings.TrimSpace(c.S3.Region)

	portNumber, err := strconv.Atoi(c.HTTP.Port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("config: APP_PORT must be a number from 1 to 65535")
	}
	if c.App.Environment == "" {
		return fmt.Errorf("config: APP_ENV must not be empty")
	}
	if c.PG.URL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("config: REDIS_ADDR must not be empty")
	}
	if c.Redis.DB < 0 {
		return fmt.Errorf("config: REDIS_DB must be a non-negative integer")
	}
	if c.Redis.DialTimeout <= 0 {
		return fmt.Errorf("config: REDIS_DIAL_TIMEOUT must be a positive duration")
	}
	if c.Redis.ReadTimeout <= 0 {
		return fmt.Errorf("config: REDIS_READ_TIMEOUT must be a positive duration")
	}
	if c.Redis.WriteTimeout <= 0 {
		return fmt.Errorf("config: REDIS_WRITE_TIMEOUT must be a positive duration")
	}
	if c.HTTP.ShutdownTimeout <= 0 {
		return fmt.Errorf("config: SHUTDOWN_TIMEOUT must be a positive duration")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("config: JWT_SECRET is required")
	}
	if c.JWT.TTL < time.Second {
		return fmt.Errorf("config: JWT_TTL must be a duration of at least 1s")
	}
	if c.Upload.Dir == "" {
		return fmt.Errorf("config: UPLOAD_DIR must not be empty")
	}
	if c.Upload.MaxFileSize <= 0 {
		return fmt.Errorf("config: MAX_UPLOAD_SIZE must be a positive integer")
	}
	if c.Upload.MaxItemPhotos <= 0 {
		return fmt.Errorf("config: MAX_ITEM_PHOTOS must be a positive integer")
	}
	switch c.Storage.Driver {
	case "local":
	case "s3":
		if c.S3.Endpoint == "" {
			return fmt.Errorf("config: S3_ENDPOINT must not be empty when STORAGE_DRIVER=s3")
		}
		if c.S3.PublicBaseURL == "" {
			return fmt.Errorf("config: S3_PUBLIC_URL must not be empty when STORAGE_DRIVER=s3")
		}
		if c.S3.Bucket == "" {
			return fmt.Errorf("config: S3_BUCKET must not be empty when STORAGE_DRIVER=s3")
		}
		if c.S3.AccessKey == "" {
			return fmt.Errorf("config: S3_ACCESS_KEY must not be empty when STORAGE_DRIVER=s3")
		}
		if c.S3.SecretKey == "" {
			return fmt.Errorf("config: S3_SECRET_KEY must not be empty when STORAGE_DRIVER=s3")
		}
		if c.S3.Region == "" {
			return fmt.Errorf("config: S3_REGION must not be empty when STORAGE_DRIVER=s3")
		}
	default:
		return fmt.Errorf("config: STORAGE_DRIVER must be one of: local, s3")
	}
	if c.Rejection.ItemTTL <= 0 {
		return fmt.Errorf("config: REJECTED_ITEM_TTL must be a positive duration")
	}

	return nil
}
