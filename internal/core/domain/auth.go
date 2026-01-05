package domain

import "time"

// Session represents an authenticated user session
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	UserAgent    string    `json:"user_agent,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// AuthContext contains authenticated user info for request context
type AuthContext struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     Role   `json:"role"`
	TeamID   string `json:"team_id"`
	SessionID string `json:"session_id"`
}

// IsAdmin checks if the authenticated user is an admin
func (a *AuthContext) IsAdmin() bool {
	return a.Role == RoleAdmin
}

// LoginRequest represents a login attempt
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is returned after successful authentication
type LoginResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         *UserSummary `json:"user"`
}

// RefreshRequest represents a token refresh attempt
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// TokenClaims represents the JWT token payload
type TokenClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      Role   `json:"role"`
	TeamID    string `json:"team_id"`
	SessionID string `json:"session_id"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetConfirm represents a password reset confirmation
type PasswordResetConfirm struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ChangePasswordRequest represents a password change by authenticated user
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
