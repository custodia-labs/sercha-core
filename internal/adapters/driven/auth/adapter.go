package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Ensure Adapter implements AuthAdapter
var _ driven.AuthAdapter = (*Adapter)(nil)

// jwtClaims wraps domain.TokenClaims for JWT compatibility
type jwtClaims struct {
	UserID    string      `json:"user_id"`
	Email     string      `json:"email"`
	Role      domain.Role `json:"role"`
	TeamID    string      `json:"team_id"`
	SessionID string      `json:"session_id"`
	jwt.RegisteredClaims
}

// Adapter handles authentication operations using bcrypt and JWT
type Adapter struct {
	jwtSecret  []byte
	bcryptCost int
}

// NewAdapter creates a new auth adapter with the given JWT secret
func NewAdapter(jwtSecret string) *Adapter {
	return &Adapter{
		jwtSecret:  []byte(jwtSecret),
		bcryptCost: bcrypt.DefaultCost,
	}
}

// NewAdapterWithCost creates a new auth adapter with custom bcrypt cost
func NewAdapterWithCost(jwtSecret string, bcryptCost int) *Adapter {
	return &Adapter{
		jwtSecret:  []byte(jwtSecret),
		bcryptCost: bcryptCost,
	}
}

// HashPassword generates a bcrypt hash from a plaintext password
func (a *Adapter) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), a.bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword checks if a password matches a bcrypt hash
func (a *Adapter) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken creates a signed JWT from domain claims
func (a *Adapter) GenerateToken(claims *domain.TokenClaims) (string, error) {
	jc := jwtClaims{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Role:      claims.Role,
		TeamID:    claims.TeamID,
		SessionID: claims.SessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Unix(claims.IssuedAt, 0)),
			ExpiresAt: jwt.NewNumericDate(time.Unix(claims.ExpiresAt, 0)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jc)
	return token.SignedString(a.jwtSecret)
}

// ParseToken validates a JWT and extracts domain claims
func (a *Adapter) ParseToken(tokenString string) (*domain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwtClaims); ok && token.Valid {
		return &domain.TokenClaims{
			UserID:    claims.UserID,
			Email:     claims.Email,
			Role:      claims.Role,
			TeamID:    claims.TeamID,
			SessionID: claims.SessionID,
			IssuedAt:  claims.IssuedAt.Unix(),
			ExpiresAt: claims.ExpiresAt.Unix(),
		}, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}
