// Package security provides password hashing and JWT primitives.
package security

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher hashes and verifies passwords with bcrypt.
type PasswordHasher struct{}

// NewPasswordHasher creates a bcrypt password hasher.
func NewPasswordHasher() PasswordHasher {
	return PasswordHasher{}
}

// Hash creates a bcrypt hash using the library's default cost.
func (PasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("generate bcrypt password hash: %w", err)
	}

	return string(hash), nil
}

// Matches reports whether a password matches a bcrypt hash.
func (PasswordHasher) Matches(passwordHash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("compare bcrypt password hash: %w", err)
	}

	return true, nil
}
