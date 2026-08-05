package user

import "time"

// RegisterRequest is the HTTP payload accepted by the registration endpoint.
type RegisterRequest struct {
	Login    string `json:"login" validate:"required" minLength:"3" maxLength:"32" example:"vadim"`
	Password string `json:"password" validate:"required" minLength:"8" example:"example-password"`
	Name     string `json:"name" validate:"required" minLength:"2" maxLength:"100" example:"Вадим"`
}

// LoginRequest is the HTTP payload accepted by the login endpoint.
type LoginRequest struct {
	Login    string `json:"login" validate:"required" example:"vadim"`
	Password string `json:"password" validate:"required" example:"example-password"`
}

// RegisterInput contains registration credentials for the service layer.
type RegisterInput struct {
	Login    string
	Password string
	Name     string
}

// LoginInput contains authentication credentials for the service layer.
type LoginInput struct {
	Login    string
	Password string
}

// PublicUser is the safe user representation returned by the API.
type PublicUser struct {
	ID           int64     `json:"id" example:"1"`
	Login        string    `json:"login" example:"vadim"`
	Name         string    `json:"name" example:"vadim"`
	PublicRating float64   `json:"public_rating" example:"0"`
	CreatedAt    time.Time `json:"created_at" example:"2026-08-04T12:00:00Z"`
}

// AuthResult contains a public user and a newly issued access token.
type AuthResult struct {
	User        PublicUser `json:"user"`
	AccessToken string     `json:"access_token" example:"<jwt_access_token>"`
	TokenType   string     `json:"token_type" example:"Bearer"`
	ExpiresIn   int64      `json:"expires_in" example:"86400"`
}
