package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

var configEnvKeys = []string{
	"APP_PORT",
	"APP_ENV",
	"DATABASE_URL",
	"REDIS_ADDR",
	"REDIS_PASSWORD",
	"REDIS_DB",
	"REDIS_DIAL_TIMEOUT",
	"REDIS_READ_TIMEOUT",
	"REDIS_WRITE_TIMEOUT",
	"REJECTED_ITEM_TTL",
	"SHUTDOWN_TIMEOUT",
	"JWT_SECRET",
	"JWT_TTL",
	"UPLOAD_DIR",
	"MAX_UPLOAD_SIZE",
	"MAX_ITEM_PHOTOS",
}

func TestLoadValidConfig(t *testing.T) {
	withEnv(t, map[string]string{
		"APP_PORT":            "8080",
		"APP_ENV":             "test",
		"DATABASE_URL":        "postgres://user:password@localhost:5432/exchange",
		"REDIS_ADDR":          "localhost:6379",
		"REDIS_PASSWORD":      "redis-secret",
		"REDIS_DB":            "1",
		"REDIS_DIAL_TIMEOUT":  "5s",
		"REDIS_READ_TIMEOUT":  "2s",
		"REDIS_WRITE_TIMEOUT": "2s",
		"REJECTED_ITEM_TTL":   "336h",
		"SHUTDOWN_TIMEOUT":    "10s",
		"JWT_SECRET":          "test-secret",
		"JWT_TTL":             "24h",
		"UPLOAD_DIR":          "/tmp/exchange-uploads",
		"MAX_UPLOAD_SIZE":     "10485760",
		"MAX_ITEM_PHOTOS":     "5",
	})

	got, err := load(false)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got.HTTP.Port != "8080" {
		t.Fatalf("HTTP.Port = %q, want %q", got.HTTP.Port, "8080")
	}
	if got.App.Environment != "test" {
		t.Fatalf("App.Environment = %q, want %q", got.App.Environment, "test")
	}
	if got.PG.URL != "postgres://user:password@localhost:5432/exchange" {
		t.Fatalf("PG.URL = %q, want configured URL", got.PG.URL)
	}
	if got.Redis.Addr != "localhost:6379" || got.Redis.DB != 1 || got.Redis.Password != "redis-secret" {
		t.Fatalf("Redis config = addr %q, db %d, password %q", got.Redis.Addr, got.Redis.DB, got.Redis.Password)
	}
	if got.Redis.DialTimeout != 5*time.Second ||
		got.Redis.ReadTimeout != 2*time.Second ||
		got.Redis.WriteTimeout != 2*time.Second {
		t.Fatalf(
			"Redis timeouts = dial %s, read %s, write %s",
			got.Redis.DialTimeout,
			got.Redis.ReadTimeout,
			got.Redis.WriteTimeout,
		)
	}
	if got.Rejection.ItemTTL != 336*time.Hour {
		t.Fatalf("Rejection.ItemTTL = %s, want %s", got.Rejection.ItemTTL, 336*time.Hour)
	}
	if got.HTTP.ShutdownTimeout != 10*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want %s", got.HTTP.ShutdownTimeout, 10*time.Second)
	}
	if got.JWT.Secret != "test-secret" {
		t.Fatalf("JWT.Secret = %q, want configured secret", got.JWT.Secret)
	}
	if got.JWT.TTL != 24*time.Hour {
		t.Fatalf("JWT.TTL = %s, want %s", got.JWT.TTL, 24*time.Hour)
	}
	if got.Upload.Dir != "/tmp/exchange-uploads" ||
		got.Upload.MaxFileSize != 10485760 ||
		got.Upload.MaxItemPhotos != 5 {
		t.Fatalf(
			"upload config = dir %q, size %d, photos %d",
			got.Upload.Dir,
			got.Upload.MaxFileSize,
			got.Upload.MaxItemPhotos,
		)
	}
}

func TestLoadUsesDefaults(t *testing.T) {
	withEnv(t, map[string]string{
		"DATABASE_URL": "postgres://user:password@localhost:5432/exchange",
		"JWT_SECRET":   "test-secret",
	})

	got, err := load(false)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got.HTTP.Port != "8080" || got.App.Environment != "local" {
		t.Fatalf("defaults = port %q, env %q", got.HTTP.Port, got.App.Environment)
	}
	if got.Redis.Addr != "redis:6379" || got.Redis.DB != 0 || got.Redis.Password != "" {
		t.Fatalf("Redis defaults = addr %q, db %d, password %q", got.Redis.Addr, got.Redis.DB, got.Redis.Password)
	}
	if got.Redis.DialTimeout != 5*time.Second ||
		got.Redis.ReadTimeout != 2*time.Second ||
		got.Redis.WriteTimeout != 2*time.Second {
		t.Fatalf(
			"Redis timeout defaults = dial %s, read %s, write %s",
			got.Redis.DialTimeout,
			got.Redis.ReadTimeout,
			got.Redis.WriteTimeout,
		)
	}
	if got.HTTP.ShutdownTimeout != 10*time.Second ||
		got.JWT.TTL != 168*time.Hour ||
		got.Rejection.ItemTTL != 336*time.Hour {
		t.Fatalf(
			"duration defaults = shutdown %s, JWT TTL %s, rejection TTL %s",
			got.HTTP.ShutdownTimeout,
			got.JWT.TTL,
			got.Rejection.ItemTTL,
		)
	}
	if got.Upload.Dir != "/app/uploads" ||
		got.Upload.MaxFileSize != 10485760 ||
		got.Upload.MaxItemPhotos != 5 {
		t.Fatalf(
			"upload defaults = dir %q, size %d, photos %d",
			got.Upload.Dir,
			got.Upload.MaxFileSize,
			got.Upload.MaxItemPhotos,
		)
	}
}

func TestLoadRejectsMissingRequiredVariables(t *testing.T) {
	tests := []struct {
		name      string
		values    map[string]string
		wantError string
	}{
		{
			name: "missing database URL",
			values: map[string]string{
				"JWT_SECRET": "test-secret",
			},
			wantError: "DATABASE_URL",
		},
		{
			name: "missing JWT secret",
			values: map[string]string{
				"DATABASE_URL": "postgres://user:password@localhost:5432/exchange",
			},
			wantError: "JWT_SECRET",
		},
		{
			name: "blank database URL",
			values: map[string]string{
				"DATABASE_URL": "  ",
				"JWT_SECRET":   "test-secret",
			},
			wantError: "DATABASE_URL",
		},
		{
			name: "blank JWT secret",
			values: map[string]string{
				"DATABASE_URL": "postgres://user:password@localhost:5432/exchange",
				"JWT_SECRET":   "  ",
			},
			wantError: "JWT_SECRET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, tt.values)

			_, err := load(false)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("load config error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		wantError string
	}{
		{name: "invalid port", key: "APP_PORT", value: "70000", wantError: "APP_PORT"},
		{name: "blank environment", key: "APP_ENV", value: " ", wantError: "APP_ENV"},
		{name: "blank Redis address", key: "REDIS_ADDR", value: " ", wantError: "REDIS_ADDR"},
		{name: "invalid Redis DB", key: "REDIS_DB", value: "cache", wantError: "REDIS_DB"},
		{name: "negative Redis DB", key: "REDIS_DB", value: "-1", wantError: "REDIS_DB"},
		{name: "invalid Redis dial timeout", key: "REDIS_DIAL_TIMEOUT", value: "soon", wantError: "REDIS_DIAL_TIMEOUT"},
		{name: "zero Redis read timeout", key: "REDIS_READ_TIMEOUT", value: "0s", wantError: "REDIS_READ_TIMEOUT"},
		{name: "invalid Redis write timeout", key: "REDIS_WRITE_TIMEOUT", value: "later", wantError: "REDIS_WRITE_TIMEOUT"},
		{name: "invalid timeout", key: "SHUTDOWN_TIMEOUT", value: "soon", wantError: "SHUTDOWN_TIMEOUT"},
		{name: "zero timeout", key: "SHUTDOWN_TIMEOUT", value: "0s", wantError: "SHUTDOWN_TIMEOUT"},
		{name: "invalid JWT TTL", key: "JWT_TTL", value: "tomorrow", wantError: "JWT_TTL"},
		{name: "zero JWT TTL", key: "JWT_TTL", value: "0s", wantError: "JWT_TTL"},
		{name: "subsecond JWT TTL", key: "JWT_TTL", value: "500ms", wantError: "JWT_TTL"},
		{name: "blank upload directory", key: "UPLOAD_DIR", value: " ", wantError: "UPLOAD_DIR"},
		{name: "invalid max upload size", key: "MAX_UPLOAD_SIZE", value: "large", wantError: "MAX_UPLOAD_SIZE"},
		{name: "zero max upload size", key: "MAX_UPLOAD_SIZE", value: "0", wantError: "MAX_UPLOAD_SIZE"},
		{name: "invalid max item photos", key: "MAX_ITEM_PHOTOS", value: "many", wantError: "MAX_ITEM_PHOTOS"},
		{name: "zero max item photos", key: "MAX_ITEM_PHOTOS", value: "0", wantError: "MAX_ITEM_PHOTOS"},
		{name: "zero rejected item TTL", key: "REJECTED_ITEM_TTL", value: "0s", wantError: "REJECTED_ITEM_TTL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{
				"DATABASE_URL": "postgres://user:password@localhost:5432/exchange",
				"JWT_SECRET":   "test-secret",
			}
			values[tt.key] = tt.value
			withEnv(t, values)

			_, err := load(false)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("load config error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func withEnv(t *testing.T, values map[string]string) {
	t.Helper()

	previous := make(map[string]string, len(configEnvKeys))
	wasSet := make(map[string]bool, len(configEnvKeys))
	for _, key := range configEnvKeys {
		value, ok := os.LookupEnv(key)
		previous[key] = value
		wasSet[key] = ok
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}

	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}

	t.Cleanup(func() {
		for _, key := range configEnvKeys {
			if !wasSet[key] {
				_ = os.Unsetenv(key)
				continue
			}
			_ = os.Setenv(key, previous[key])
		}
	})
}
