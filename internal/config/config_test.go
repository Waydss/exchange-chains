package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadValidConfig(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"APP_PORT":         "8080",
		"APP_ENV":          "test",
		"DATABASE_URL":     "postgres://user:password@localhost:5432/exchange",
		"SHUTDOWN_TIMEOUT": "10s",
		"JWT_SECRET":       "test-secret",
		"JWT_TTL":          "24h",
		"UPLOAD_DIR":       "/tmp/exchange-uploads",
		"MAX_UPLOAD_SIZE":  "10485760",
		"MAX_ITEM_PHOTOS":  "5",
	}

	got, err := load(mapLookup(values))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got.Port != "8080" {
		t.Fatalf("Port = %q, want %q", got.Port, "8080")
	}
	if got.Environment != "test" {
		t.Fatalf("Environment = %q, want %q", got.Environment, "test")
	}
	if got.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want %s", got.ShutdownTimeout, 10*time.Second)
	}
	if got.JWTSecret != "test-secret" {
		t.Fatalf("JWTSecret = %q, want configured secret", got.JWTSecret)
	}
	if got.JWTTTL != 24*time.Hour {
		t.Fatalf("JWTTTL = %s, want %s", got.JWTTTL, 24*time.Hour)
	}
	if got.UploadDir != "/tmp/exchange-uploads" || got.MaxUploadSize != 10485760 || got.MaxItemPhotos != 5 {
		t.Fatalf("upload config = dir %q, size %d, photos %d", got.UploadDir, got.MaxUploadSize, got.MaxItemPhotos)
	}
}

func TestLoadRejectsMissingVariable(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"APP_PORT":         "8080",
		"APP_ENV":          "test",
		"SHUTDOWN_TIMEOUT": "10s",
		"JWT_SECRET":       "test-secret",
		"JWT_TTL":          "24h",
		"UPLOAD_DIR":       "/tmp/exchange-uploads",
		"MAX_UPLOAD_SIZE":  "10485760",
		"MAX_ITEM_PHOTOS":  "5",
	}

	_, err := load(mapLookup(values))
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("load config error = %v, want missing DATABASE_URL error", err)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		value     string
		wantError string
	}{
		{name: "invalid port", key: "APP_PORT", value: "70000", wantError: "APP_PORT"},
		{name: "invalid timeout", key: "SHUTDOWN_TIMEOUT", value: "soon", wantError: "SHUTDOWN_TIMEOUT"},
		{name: "zero timeout", key: "SHUTDOWN_TIMEOUT", value: "0s", wantError: "SHUTDOWN_TIMEOUT"},
		{name: "invalid JWT TTL", key: "JWT_TTL", value: "tomorrow", wantError: "JWT_TTL"},
		{name: "zero JWT TTL", key: "JWT_TTL", value: "0s", wantError: "JWT_TTL"},
		{name: "subsecond JWT TTL", key: "JWT_TTL", value: "500ms", wantError: "JWT_TTL"},
		{name: "invalid max upload size", key: "MAX_UPLOAD_SIZE", value: "large", wantError: "MAX_UPLOAD_SIZE"},
		{name: "zero max upload size", key: "MAX_UPLOAD_SIZE", value: "0", wantError: "MAX_UPLOAD_SIZE"},
		{name: "invalid max item photos", key: "MAX_ITEM_PHOTOS", value: "many", wantError: "MAX_ITEM_PHOTOS"},
		{name: "zero max item photos", key: "MAX_ITEM_PHOTOS", value: "0", wantError: "MAX_ITEM_PHOTOS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			values := map[string]string{
				"APP_PORT":         "8080",
				"APP_ENV":          "test",
				"DATABASE_URL":     "postgres://user:password@localhost:5432/exchange",
				"SHUTDOWN_TIMEOUT": "10s",
				"JWT_SECRET":       "test-secret",
				"JWT_TTL":          "24h",
				"UPLOAD_DIR":       "/tmp/exchange-uploads",
				"MAX_UPLOAD_SIZE":  "10485760",
				"MAX_ITEM_PHOTOS":  "5",
			}
			values[tt.key] = tt.value

			_, err := load(mapLookup(values))
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("load config error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func TestLoadRejectsMissingJWTConfig(t *testing.T) {
	t.Parallel()

	base := map[string]string{
		"APP_PORT":         "8080",
		"APP_ENV":          "test",
		"DATABASE_URL":     "postgres://user:password@localhost:5432/exchange",
		"SHUTDOWN_TIMEOUT": "10s",
		"JWT_TTL":          "24h",
		"UPLOAD_DIR":       "/tmp/exchange-uploads",
		"MAX_UPLOAD_SIZE":  "10485760",
		"MAX_ITEM_PHOTOS":  "5",
	}

	_, err := load(mapLookup(base))
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET is required") {
		t.Fatalf("load config error = %v, want missing JWT_SECRET error", err)
	}

	base["JWT_SECRET"] = "test-secret"
	delete(base, "JWT_TTL")
	_, err = load(mapLookup(base))
	if err == nil || !strings.Contains(err.Error(), "JWT_TTL is required") {
		t.Fatalf("load config error = %v, want missing JWT_TTL error", err)
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
