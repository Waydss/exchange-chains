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
	Port              string
	Environment       string
	DatabaseURL       string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	RedisDialTimeout  time.Duration
	RedisReadTimeout  time.Duration
	RedisWriteTimeout time.Duration
	RejectedItemTTL   time.Duration
	ShutdownTimeout   time.Duration
	JWTSecret         string
	JWTTTL            time.Duration
	UploadDir         string
	MaxUploadSize     int64
	MaxItemPhotos     int
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

	redisAddr, err := required(lookup, "REDIS_ADDR")
	if err != nil {
		return Config{}, err
	}

	redisDBValue, err := required(lookup, "REDIS_DB")
	if err != nil {
		return Config{}, err
	}
	redisDB, err := strconv.Atoi(redisDBValue)
	if err != nil || redisDB < 0 {
		return Config{}, fmt.Errorf("config: REDIS_DB must be a non-negative integer")
	}

	redisDialTimeout, err := positiveDuration(lookup, "REDIS_DIAL_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	redisReadTimeout, err := positiveDuration(lookup, "REDIS_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	redisWriteTimeout, err := positiveDuration(lookup, "REDIS_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	rejectedItemTTL, err := positiveDuration(lookup, "REJECTED_ITEM_TTL")
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
		Port:              port,
		Environment:       environment,
		DatabaseURL:       databaseURL,
		RedisAddr:         redisAddr,
		RedisPassword:     optional(lookup, "REDIS_PASSWORD"),
		RedisDB:           redisDB,
		RedisDialTimeout:  redisDialTimeout,
		RedisReadTimeout:  redisReadTimeout,
		RedisWriteTimeout: redisWriteTimeout,
		RejectedItemTTL:   rejectedItemTTL,
		ShutdownTimeout:   shutdownTimeout,
		JWTSecret:         jwtSecret,
		JWTTTL:            jwtTTL,
		UploadDir:         uploadDir,
		MaxUploadSize:     maxUploadSize,
		MaxItemPhotos:     maxItemPhotos,
	}, nil
}

func positiveDuration(lookup func(string) (string, bool), key string) (time.Duration, error) {
	value, err := required(lookup, key)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("config: %s must be a positive duration", key)
	}

	return duration, nil
}

func optional(lookup func(string) (string, bool), key string) string {
	value, _ := lookup(key)
	return value
}

func required(lookup func(string) (string, bool), key string) (string, error) {
	value, ok := lookup(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return "", fmt.Errorf("config: %s is required", key)
	}

	return value, nil
}
