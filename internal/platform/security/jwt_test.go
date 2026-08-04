package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTGenerateAndParse(t *testing.T) {
	now := time.Date(2026, time.August, 3, 17, 0, 0, 0, time.UTC)
	provider, err := newJWT("test-secret", 24*time.Hour, func() time.Time { return now })
	if err != nil {
		t.Fatalf("create JWT provider: %v", err)
	}

	token, err := provider.Generate(42, "vadim")
	if err != nil {
		t.Fatalf("generate JWT: %v", err)
	}

	claims, err := provider.Parse(token)
	if err != nil {
		t.Fatalf("parse JWT: %v", err)
	}
	if claims.UserID != 42 || claims.Login != "vadim" {
		t.Fatalf("claims = %+v, want user 42 with login vadim", claims)
	}
}

func TestJWTRejectsExpiredToken(t *testing.T) {
	issuedAt := time.Date(2026, time.August, 3, 17, 0, 0, 0, time.UTC)
	currentTime := issuedAt
	provider, err := newJWT("test-secret", time.Hour, func() time.Time { return currentTime })
	if err != nil {
		t.Fatalf("create JWT provider: %v", err)
	}

	token, err := provider.Generate(42, "vadim")
	if err != nil {
		t.Fatalf("generate JWT: %v", err)
	}

	currentTime = issuedAt.Add(2 * time.Hour)
	if _, err := provider.Parse(token); err == nil {
		t.Fatal("Parse() error = nil, want expired token error")
	}
}

func TestJWTRejectsWrongSignature(t *testing.T) {
	provider, err := NewJWT("correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("create JWT provider: %v", err)
	}
	otherProvider, err := NewJWT("wrong-secret", time.Hour)
	if err != nil {
		t.Fatalf("create second JWT provider: %v", err)
	}

	token, err := provider.Generate(42, "vadim")
	if err != nil {
		t.Fatalf("generate JWT: %v", err)
	}

	if _, err := otherProvider.Parse(token); err == nil {
		t.Fatal("Parse() error = nil, want invalid signature error")
	}
}

func TestJWTRejectsUnexpectedAlgorithm(t *testing.T) {
	now := time.Date(2026, time.August, 3, 17, 0, 0, 0, time.UTC)
	provider, err := newJWT("test-secret", time.Hour, func() time.Time { return now })
	if err != nil {
		t.Fatalf("create JWT provider: %v", err)
	}

	claims := validTokenClaims(now)
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	signedToken, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign HS384 JWT: %v", err)
	}

	if _, err := provider.Parse(signedToken); err == nil {
		t.Fatal("Parse() error = nil, want unexpected algorithm error")
	}
}

func TestJWTRejectsInvalidClaims(t *testing.T) {
	now := time.Date(2026, time.August, 3, 17, 0, 0, 0, time.UTC)
	provider, err := newJWT("test-secret", time.Hour, func() time.Time { return now })
	if err != nil {
		t.Fatalf("create JWT provider: %v", err)
	}

	tests := []struct {
		name   string
		claims tokenClaims
	}{
		{
			name: "missing issued at",
			claims: tokenClaims{
				Login: "vadim",
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "42",
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
				},
			},
		},
		{
			name: "future issued at",
			claims: tokenClaims{
				Login: "vadim",
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "42",
					IssuedAt:  jwt.NewNumericDate(now.Add(time.Minute)),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
				},
			},
		},
		{
			name: "invalid subject",
			claims: tokenClaims{
				Login: "vadim",
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "not-an-ID",
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
				},
			},
		},
		{
			name: "missing login",
			claims: tokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "42",
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, tt.claims)
			signedToken, err := token.SignedString([]byte("test-secret"))
			if err != nil {
				t.Fatalf("sign JWT: %v", err)
			}

			if _, err := provider.Parse(signedToken); err == nil {
				t.Fatal("Parse() error = nil, want invalid claims error")
			}
		})
	}
}

func TestJWTRejectsInvalidGenerationInput(t *testing.T) {
	provider, err := NewJWT("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("create JWT provider: %v", err)
	}

	tests := []struct {
		name   string
		userID int64
		login  string
	}{
		{name: "zero user ID", login: "vadim"},
		{name: "empty login", userID: 42, login: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := provider.Generate(tt.userID, tt.login); err == nil {
				t.Fatal("Generate() error = nil, want validation error")
			}
		})
	}
}

func TestNewJWTRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		ttl    time.Duration
	}{
		{name: "empty secret", ttl: time.Hour},
		{name: "subsecond TTL", secret: "test-secret", ttl: 500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewJWT(tt.secret, tt.ttl); err == nil {
				t.Fatal("NewJWT() error = nil, want configuration error")
			}
		})
	}
}

func validTokenClaims(now time.Time) tokenClaims {
	return tokenClaims{
		Login: "vadim",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "42",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
}
