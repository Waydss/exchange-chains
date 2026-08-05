// Package config loads and validates application configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains validated runtime settings.
type Config struct {
	Port            string
	Environment     string
	DatabaseURL     string
	ShutdownTimeout time.Duration
	JWTSecret       string
	JWTTTL          time.Duration
	UploadDir       string
	MaxUploadSize   int64
	MaxItemPhotos   int
}

// Load reads and validates the application configuration.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup func(string) (string, bool)) (Config, error) {
	port, err := required(lookup, "APP_PORT")
	if err != nil {
		return Config{}, err
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return Config{}, fmt.Errorf("config: APP_PORT must be a number from 1 to 65535")
	}

	environment, err := required(lookup, "APP_ENV")
	if err != nil {
		return Config{}, err
	}

	databaseURL, err := required(lookup, "DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	shutdownTimeoutValue, err := required(lookup, "SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := time.ParseDuration(shutdownTimeoutValue)
	if err != nil || shutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("config: SHUTDOWN_TIMEOUT must be a positive duration")
	}

	jwtSecret, err := required(lookup, "JWT_SECRET")
	if err != nil {
		return Config{}, err
	}

	jwtTTLValue, err := required(lookup, "JWT_TTL")
	if err != nil {
		return Config{}, err
	}

	jwtTTL, err := time.ParseDuration(jwtTTLValue)
	if err != nil || jwtTTL < time.Second {
		return Config{}, fmt.Errorf("config: JWT_TTL must be a duration of at least 1s")
	}

	uploadDir, err := required(lookup, "UPLOAD_DIR")
	if err != nil {
		return Config{}, err
	}

	maxUploadSizeValue, err := required(lookup, "MAX_UPLOAD_SIZE")
	if err != nil {
		return Config{}, err
	}
	maxUploadSize, err := strconv.ParseInt(maxUploadSizeValue, 10, 64)
	if err != nil || maxUploadSize <= 0 {
		return Config{}, fmt.Errorf("config: MAX_UPLOAD_SIZE must be a positive integer")
	}

	maxItemPhotosValue, err := required(lookup, "MAX_ITEM_PHOTOS")
	if err != nil {
		return Config{}, err
	}
	maxItemPhotos, err := strconv.Atoi(maxItemPhotosValue)
	if err != nil || maxItemPhotos <= 0 {
		return Config{}, fmt.Errorf("config: MAX_ITEM_PHOTOS must be a positive integer")
	}

	return Config{
		Port:            port,
		Environment:     environment,
		DatabaseURL:     databaseURL,
		ShutdownTimeout: shutdownTimeout,
		JWTSecret:       jwtSecret,
		JWTTTL:          jwtTTL,
		UploadDir:       uploadDir,
		MaxUploadSize:   maxUploadSize,
		MaxItemPhotos:   maxItemPhotos,
	}, nil
}

func required(lookup func(string) (string, bool), key string) (string, error) {
	value, ok := lookup(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return "", fmt.Errorf("config: %s is required", key)
	}

	return value, nil
}
