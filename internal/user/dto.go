package user

import "time"

// RegisterRequest is the HTTP payload accepted by the registration endpoint.
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// LoginRequest is the HTTP payload accepted by the login endpoint.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// RegisterInput contains registration credentials for the service layer.
type RegisterInput struct {
	Login    string
	Password string
}

// LoginInput contains authentication credentials for the service layer.
type LoginInput struct {
	Login    string
	Password string
}

// PublicUser is the safe user representation returned by the API.
type PublicUser struct {
	ID           int64     `json:"id"`
	Login        string    `json:"login"`
	Name         string    `json:"name"`
	PublicRating float64   `json:"public_rating"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuthResult contains a public user and a newly issued access token.
type AuthResult struct {
	User        PublicUser `json:"user"`
	AccessToken string     `json:"access_token"`
	TokenType   string     `json:"token_type"`
	ExpiresIn   int64      `json:"expires_in"`
}
