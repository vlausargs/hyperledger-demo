package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// Hardcoded credentials live here (matching the pre-refactor behaviour).
// A future ticket replaces them with a real user store.
const (
	hardcodedUser = "admin"
	hardcodedPass = "asdqwe123"
	tokenTTL      = 15 * time.Minute
)

// LoginResult is the payload returned to a successful login.
type LoginResult struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

// AuthService issues JWTs for valid credentials.
type AuthService struct {
	jwtSecret string
	mspID     string
}

func NewAuthService(jwtSecret, mspID string) *AuthService {
	return &AuthService{jwtSecret: jwtSecret, mspID: mspID}
}

// Login validates the username/password and returns a signed JWT.
// Returns ErrInternal if the server has no JWT secret configured, and
// ErrUnauthorized on bad creds.
func (s *AuthService) Login(_ context.Context, username, password string) (*LoginResult, *apperrors.AppError) {
	if s.jwtSecret == "" {
		return nil, apperrors.NewInternal("server misconfigured: JWT secret not set", "")
	}
	if username != hardcodedUser || password != hardcodedPass {
		return nil, apperrors.NewUnauthorized("invalid credentials")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  username,
		"org":  s.mspID,
		"role": "admin",
		"exp":  time.Now().Add(tokenTTL).Unix(),
		"iat":  time.Now().Unix(),
	})
	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, apperrors.NewInternal("failed to generate token", err.Error())
	}
	return &LoginResult{Token: signed, ExpiresIn: int(tokenTTL.Seconds())}, nil
}
