package security

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims contains authenticated identity data extracted from an access token.
type Claims struct {
	UserID int64
	Login  string
}

type tokenClaims struct {
	Login string `json:"login"`
	jwt.RegisteredClaims
}

// JWT manages signed HS256 access tokens.
type JWT struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewJWT creates an HS256 access-token provider.
func NewJWT(secret string, ttl time.Duration) (*JWT, error) {
	return newJWT(secret, ttl, time.Now)
}

func newJWT(secret string, ttl time.Duration, now func() time.Time) (*JWT, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("create JWT provider: secret must not be empty")
	}
	if ttl < time.Second {
		return nil, fmt.Errorf("create JWT provider: TTL must be at least 1s")
	}

	return &JWT{
		secret: []byte(secret),
		ttl:    ttl,
		now:    now,
	}, nil
}

// Generate creates a signed access token for a user.
func (j *JWT) Generate(userID int64, login string) (string, error) {
	if userID <= 0 {
		return "", fmt.Errorf("generate JWT access token: user ID must be positive")
	}
	if strings.TrimSpace(login) == "" {
		return "", fmt.Errorf("generate JWT access token: login must not be empty")
	}

	now := j.now().UTC()
	claims := tokenClaims{
		Login: login,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("sign JWT access token: %w", err)
	}

	return signedToken, nil
}

// Parse verifies an access token and returns its identity claims.
func (j *JWT) Parse(rawToken string) (Claims, error) {
	claims := tokenClaims{}
	token, err := jwt.ParseWithClaims(
		rawToken,
		&claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected JWT signing algorithm: %s", token.Method.Alg())
			}
			return j.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(j.now),
	)
	if err != nil {
		return Claims{}, fmt.Errorf("parse JWT access token: %w", err)
	}
	if !token.Valid || claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return Claims{}, fmt.Errorf("parse JWT access token: required claims are missing")
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return Claims{}, fmt.Errorf("parse JWT access token subject: invalid user ID")
	}
	if strings.TrimSpace(claims.Login) == "" {
		return Claims{}, fmt.Errorf("parse JWT access token: login claim is missing")
	}

	return Claims{UserID: userID, Login: claims.Login}, nil
}

// TTL returns the configured access-token lifetime.
func (j *JWT) TTL() time.Duration {
	return j.ttl
}
