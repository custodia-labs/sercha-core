package domain

import "time"

// Role defines user permission level
type Role string

const (
	RoleAdmin  Role = "admin"  // Manage users, sources, settings
	RoleMember Role = "member" // Search, view documents
	RoleViewer Role = "viewer" // Search only (future)
)

// User represents a team member
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialize
	Name         string    `json:"name"`
	Role         Role      `json:"role"`
	TeamID       string    `json:"team_id"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// Team represents an organization/team
type Team struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserSummary provides a safe view of user data (no password hash)
type UserSummary struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Role        Role       `json:"role"`
	Active      bool       `json:"active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// ToSummary converts a User to UserSummary
func (u *User) ToSummary() *UserSummary {
	return &UserSummary{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		Role:        u.Role,
		Active:      u.Active,
		LastLoginAt: u.LastLoginAt,
	}
}

// IsAdmin checks if the user has admin privileges
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// CanManageUsers checks if the user can create/delete other users
func (u *User) CanManageUsers() bool {
	return u.Role == RoleAdmin
}

// CanManageSources checks if the user can create/delete sources
func (u *User) CanManageSources() bool {
	return u.Role == RoleAdmin
}

// CanSearch checks if the user can perform searches
func (u *User) CanSearch() bool {
	return u.Active && (u.Role == RoleAdmin || u.Role == RoleMember || u.Role == RoleViewer)
}
