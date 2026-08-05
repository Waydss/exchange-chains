package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/platform/security"
	"github.com/L-ka080/DDB-6-case/internal/user"
)

type fakeRepository struct {
	usersByLogin map[string]user.User
	usersByID    map[int64]user.User
	nextID       int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		usersByLogin: make(map[string]user.User),
		usersByID:    make(map[int64]user.User),
		nextID:       1,
	}
}

func (r *fakeRepository) Create(_ context.Context, params user.CreateUserParams) (user.User, error) {
	if _, exists := r.usersByLogin[params.Login]; exists {
		return user.User{}, user.ErrLoginAlreadyExists
	}

	createdUser := user.User{
		ID:           r.nextID,
		Login:        params.Login,
		PasswordHash: params.PasswordHash,
		Name:         params.Name,
		CreatedAt:    time.Date(2026, time.August, 3, 17, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, time.August, 3, 17, 0, 0, 0, time.UTC),
	}
	r.nextID++
	r.usersByLogin[createdUser.Login] = createdUser
	r.usersByID[createdUser.ID] = createdUser

	return createdUser, nil
}

func (r *fakeRepository) GetByLogin(_ context.Context, login string) (user.User, error) {
	foundUser, ok := r.usersByLogin[login]
	if !ok {
		return user.User{}, user.ErrUserNotFound
	}
	return foundUser, nil
}

func (r *fakeRepository) GetByID(_ context.Context, id int64) (user.User, error) {
	foundUser, ok := r.usersByID[id]
	if !ok {
		return user.User{}, user.ErrUserNotFound
	}
	return foundUser, nil
}

type fakeTokenProvider struct{}

func (fakeTokenProvider) Generate(_ int64, _ string) (string, error) {
	return "access-token", nil
}

func (fakeTokenProvider) TTL() time.Duration {
	return 24 * time.Hour
}

func TestRegisterSuccess(t *testing.T) {
	repository := newFakeRepository()
	service := newService(repository)

	result, err := service.Register(context.Background(), user.RegisterInput{
		Login:    "  Vadim  ",
		Password: "password123",
		Name:     "  Вадим  ",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if result.User.Login != "vadim" {
		t.Fatalf("login = %q, want normalized login", result.User.Login)
	}
	if result.User.Name != "Вадим" {
		t.Fatalf("name = %q, want normalized name", result.User.Name)
	}
	if result.AccessToken != "access-token" || result.TokenType != "Bearer" || result.ExpiresIn != 86400 {
		t.Fatalf("auth result = %+v", result)
	}

	storedUser := repository.usersByLogin["vadim"]
	if storedUser.PasswordHash == "password123" || storedUser.PasswordHash == "" {
		t.Fatal("repository received an unsafe password hash")
	}
}

func TestRegisterRejectsOccupiedLogin(t *testing.T) {
	repository := newFakeRepository()
	repository.usersByLogin["vadim"] = user.User{ID: 1, Login: "vadim"}
	service := newService(repository)

	_, err := service.Register(context.Background(), user.RegisterInput{
		Login:    "VADIM",
		Password: "password123",
		Name:     "Вадим",
	})
	if !errors.Is(err, user.ErrLoginAlreadyExists) {
		t.Fatalf("Register() error = %v, want ErrLoginAlreadyExists", err)
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	service := newService(newFakeRepository())

	_, err := service.Register(context.Background(), user.RegisterInput{
		Login:    "vadim",
		Password: "short",
		Name:     "Вадим",
	})
	assertValidationError(t, err)
}

func TestRegisterRejectsPasswordOverBcryptLimit(t *testing.T) {
	service := newService(newFakeRepository())

	_, err := service.Register(context.Background(), user.RegisterInput{
		Login:    "vadim",
		Password: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Name:     "Вадим",
	})
	assertValidationError(t, err)
}

func TestRegisterRejectsInvalidLogin(t *testing.T) {
	service := newService(newFakeRepository())

	_, err := service.Register(context.Background(), user.RegisterInput{
		Login:    "bad login",
		Password: "password123",
		Name:     "Вадим",
	})
	assertValidationError(t, err)
}

func TestRegisterRejectsEmptyName(t *testing.T) {
	service := newService(newFakeRepository())

	_, err := service.Register(context.Background(), user.RegisterInput{
		Login:    "vadim",
		Password: "password123",
		Name:     "   ",
	})
	assertValidationError(t, err)
}

func TestLoginSuccess(t *testing.T) {
	repository := newFakeRepository()
	passwordHash := mustHashPassword(t, "password123")
	repository.usersByLogin["vadim"] = user.User{
		ID:           7,
		Login:        "vadim",
		PasswordHash: passwordHash,
	}
	service := newService(repository)

	result, err := service.Login(context.Background(), user.LoginInput{
		Login:    " Vadim ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.User.ID != 7 || result.User.Login != "vadim" || result.AccessToken != "access-token" {
		t.Fatalf("auth result = %+v", result)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	repository := newFakeRepository()
	repository.usersByLogin["vadim"] = user.User{
		ID:           7,
		Login:        "vadim",
		PasswordHash: mustHashPassword(t, "password123"),
	}
	service := newService(repository)

	_, err := service.Login(context.Background(), user.LoginInput{
		Login:    "vadim",
		Password: "wrong-password",
	})
	if !errors.Is(err, user.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginHidesUnknownUser(t *testing.T) {
	service := newService(newFakeRepository())

	_, err := service.Login(context.Background(), user.LoginInput{
		Login:    "unknown",
		Password: "password123",
	})
	if !errors.Is(err, user.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginRejectsMissingCredentials(t *testing.T) {
	service := newService(newFakeRepository())

	tests := []struct {
		name  string
		input user.LoginInput
	}{
		{name: "missing login", input: user.LoginInput{Password: "password123"}},
		{name: "missing password", input: user.LoginInput{Login: "vadim"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Login(context.Background(), tt.input)
			assertValidationError(t, err)
		})
	}
}

func TestGetByIDReturnsNotFound(t *testing.T) {
	service := newService(newFakeRepository())

	_, err := service.GetByID(context.Background(), 999)
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrUserNotFound", err)
	}
}

func newService(repository user.Repository) *user.Service {
	return user.NewService(repository, security.NewPasswordHasher(), fakeTokenProvider{})
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := security.NewPasswordHasher().Hash(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return hash
}

func assertValidationError(t *testing.T, err error) {
	t.Helper()
	var validationError *user.ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
}
