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
	minNameLength     = 2
	maxNameLength     = 100
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
	name := strings.TrimSpace(input.Name)
	if err := validateLogin(login); err != nil {
		return AuthResult{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return AuthResult{}, err
	}
	if err := validateName(name); err != nil {
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
		Name:         name,
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

// Returns the publicRating and shadowRating for the user, found by provided UserID. These will be used in construction for the userCandidate in chain_builder package
func (s *Service) GetUserCandidateBase(ctx context.Context, userId int64) (publicRating float64, shadowRating int, err error) {
	foundUser, err := s.repository.GetByID(ctx, userId)
	if err != nil {
		return 0, 0, fmt.Errorf("load user profile: %w", err)
	}

	return foundUser.PublicRating, foundUser.ShadowRating, nil
}

// Called by separate handler, when one of the users decided to leave a rating after closed barter between each other.
//
// Each user in the barter can leave a rating for other
func (s *Service) UpdatePublicRating(ctx context.Context, fromId int64, toId int64, rating float64, barterId int64) (TotalUserRatingDTO, error) {
	if rating < 0 || rating > 5 {
		return TotalUserRatingDTO{}, &ValidationError{Message: "rating must be between 0 and 5"}
	}

	if fromId <= 0 || toId <= 0 {
		return TotalUserRatingDTO{}, &ValidationError{Message: "from and to user IDs must be positive"}
	}

	if _, err := s.repository.GetByID(ctx, fromId); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return TotalUserRatingDTO{}, fmt.Errorf("from user: %w", err)
		}
		return TotalUserRatingDTO{}, fmt.Errorf("load from user: %w", err)
	}

	if _, err := s.repository.GetByID(ctx, toId); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return TotalUserRatingDTO{}, fmt.Errorf("to user: %w", err)
		}
		return TotalUserRatingDTO{}, fmt.Errorf("load to user: %w", err)
	}

	if err := s.repository.ValidateBarterParticipants(ctx, barterId, fromId, toId); err != nil {
		return TotalUserRatingDTO{}, err
	}

	average, total, err := s.repository.UpdatePublicRating(ctx, fromId, toId, rating, barterId)
	if err != nil {
		return TotalUserRatingDTO{}, fmt.Errorf("update public rating: %w", err)
	}

	return TotalUserRatingDTO{
		UserRating:  average,
		TotalScores: total,
	}, nil
}

// GetUserRatings returns the summary of the current public rating for the user,
// together with every rating record where the user is the recipient (to_id).
func (s *Service) GetUserRatings(ctx context.Context, userId int64) (UserRatingsDTO, error) {
	if userId <= 0 {
		return UserRatingsDTO{}, &ValidationError{Message: "user ID must be positive"}
	}

	ratings, err := s.repository.GetUserRatings(ctx, userId)
	if err != nil {
		return UserRatingsDTO{}, fmt.Errorf("load user ratings: %w", err)
	}

	total := len(ratings)
	entries := make([]UserRatingEntry, 0, total)
	var sum float64
	for _, rating := range ratings {
		fromUser, err := s.repository.GetByID(ctx, rating.FromId)
		if err != nil {
			return UserRatingsDTO{}, fmt.Errorf("load rater profile: %w", err)
		}

		sum += rating.PublicRating
		entries = append(entries, UserRatingEntry{
			User:     toPublicUser(fromUser),
			Rating:   rating.PublicRating,
			BarterId: rating.BarterId,
		})
	}

	var average float64
	if total > 0 {
		average = sum / float64(total)
	}

	return UserRatingsDTO{
		UserRating:  average,
		TotalScores: total,
		Ratings:     entries,
	}, nil
}

func (s *Service) UpdateShadowRating(ctx context.Context, userId int64, reasonString string) error {
	s.repository.UpdateShadowRating(ctx, userId, reasonString)

	return fmt.Errorf("[user/service] Not implemented")
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

func validateName(name string) error {
	length := utf8.RuneCountInString(name)
	if length < minNameLength || length > maxNameLength {
		return &ValidationError{Message: "name must contain between 2 and 100 characters"}
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
