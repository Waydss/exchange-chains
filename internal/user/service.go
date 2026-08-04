package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	minLoginLength    = 3
	maxLoginLength    = 32
	minPasswordLength = 8
	maxPasswordBytes  = 72
)

var loginPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]*$`)

// PasswordHasher is the password capability required by the user service.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Matches(passwordHash, password string) (bool, error)
}

// TokenProvider is the access-token capability required by the user service.
type TokenProvider interface {
	Generate(userID int64, login string) (string, error)
	TTL() time.Duration
}

// Service implements registration, authentication, and profile retrieval.
type Service struct {
	repository     Repository
	passwordHasher PasswordHasher
	tokenProvider  TokenProvider
}

// NewService constructs the user service with explicit dependencies.
func NewService(repository Repository, passwordHasher PasswordHasher, tokenProvider TokenProvider) *Service {
	return &Service{
		repository:     repository,
		passwordHasher: passwordHasher,
		tokenProvider:  tokenProvider,
	}
}

// Register creates an account and returns a new access token.
func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	login := normalizeLogin(input.Login)
	if err := validateLogin(login); err != nil {
		return AuthResult{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return AuthResult{}, err
	}

	_, err := s.repository.GetByLogin(ctx, login)
	if err == nil {
		return AuthResult{}, ErrLoginAlreadyExists
	}
	if !errors.Is(err, ErrUserNotFound) {
		return AuthResult{}, fmt.Errorf("check login availability: %w", err)
	}

	passwordHash, err := s.passwordHasher.Hash(input.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash user password: %w", err)
	}

	createdUser, err := s.repository.Create(ctx, CreateUserParams{
		Login:        login,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return AuthResult{}, fmt.Errorf("persist registered user: %w", err)
	}

	return s.authResult(createdUser)
}

// Login verifies credentials and returns a new access token.
func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	login := normalizeLogin(input.Login)
	if login == "" || input.Password == "" {
		return AuthResult{}, &ValidationError{Message: "login and password are required"}
	}

	foundUser, err := s.repository.GetByLogin(ctx, login)
	if errors.Is(err, ErrUserNotFound) {
		return AuthResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, fmt.Errorf("load user for login: %w", err)
	}

	matches, err := s.passwordHasher.Matches(foundUser.PasswordHash, input.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("verify user password: %w", err)
	}
	if !matches {
		return AuthResult{}, ErrInvalidCredentials
	}

	return s.authResult(foundUser)
}

// GetByID returns the public representation of a user.
func (s *Service) GetByID(ctx context.Context, userID int64) (PublicUser, error) {
	foundUser, err := s.repository.GetByID(ctx, userID)
	if err != nil {
		return PublicUser{}, fmt.Errorf("load user profile: %w", err)
	}

	return toPublicUser(foundUser), nil
}

func (s *Service) authResult(account User) (AuthResult, error) {
	accessToken, err := s.tokenProvider.Generate(account.ID, account.Login)
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate access token: %w", err)
	}

	return AuthResult{
		User:        toPublicUser(account),
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.tokenProvider.TTL().Seconds()),
	}, nil
}

func normalizeLogin(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}

func validateLogin(login string) error {
	length := utf8.RuneCountInString(login)
	if length < minLoginLength || length > maxLoginLength {
		return &ValidationError{Message: "login must contain between 3 and 32 characters"}
	}
	if !loginPattern.MatchString(login) {
		return &ValidationError{
			Message: "login may contain only latin letters, digits, and underscore and must start with a letter or digit",
		}
	}

	return nil
}

func validatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return &ValidationError{Message: "password must not contain only spaces"}
	}
	if utf8.RuneCountInString(password) < minPasswordLength {
		return &ValidationError{Message: "password must contain at least 8 characters"}
	}
	if len(password) > maxPasswordBytes {
		return &ValidationError{Message: "password must not exceed 72 bytes"}
	}

	return nil
}

func toPublicUser(account User) PublicUser {
	return PublicUser{
		ID:           account.ID,
		Login:        account.Login,
		Name:         account.Name,
		PublicRating: account.PublicRating,
		CreatedAt:    account.CreatedAt,
	}
}
