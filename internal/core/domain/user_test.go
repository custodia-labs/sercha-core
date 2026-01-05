package domain

import (
	"testing"
	"time"
)

func TestUserToSummary(t *testing.T) {
	now := time.Now()
	user := &User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: "secret-hash",
		Name:         "Test User",
		Role:         RoleAdmin,
		TeamID:       "team-123",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastLoginAt:  &now,
	}

	summary := user.ToSummary()

	if summary.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, summary.ID)
	}
	if summary.Email != user.Email {
		t.Errorf("expected Email %s, got %s", user.Email, summary.Email)
	}
	if summary.Name != user.Name {
		t.Errorf("expected Name %s, got %s", user.Name, summary.Name)
	}
	if summary.Role != user.Role {
		t.Errorf("expected Role %s, got %s", user.Role, summary.Role)
	}
	if summary.Active != user.Active {
		t.Errorf("expected Active %v, got %v", user.Active, summary.Active)
	}
	if summary.LastLoginAt == nil {
		t.Error("expected LastLoginAt to be set")
	}
}

func TestUserIsAdmin(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleAdmin, true},
		{RoleMember, false},
		{RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			user := &User{Role: tt.role}
			if user.IsAdmin() != tt.expected {
				t.Errorf("expected IsAdmin() = %v for role %s", tt.expected, tt.role)
			}
		})
	}
}

func TestUserCanManageUsers(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleAdmin, true},
		{RoleMember, false},
		{RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			user := &User{Role: tt.role}
			if user.CanManageUsers() != tt.expected {
				t.Errorf("expected CanManageUsers() = %v for role %s", tt.expected, tt.role)
			}
		})
	}
}

func TestUserCanManageSources(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleAdmin, true},
		{RoleMember, false},
		{RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			user := &User{Role: tt.role}
			if user.CanManageSources() != tt.expected {
				t.Errorf("expected CanManageSources() = %v for role %s", tt.expected, tt.role)
			}
		})
	}
}

func TestUserCanSearch(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		active   bool
		expected bool
	}{
		{"active admin", RoleAdmin, true, true},
		{"active member", RoleMember, true, true},
		{"active viewer", RoleViewer, true, true},
		{"inactive admin", RoleAdmin, false, false},
		{"inactive member", RoleMember, false, false},
		{"inactive viewer", RoleViewer, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.role, Active: tt.active}
			if user.CanSearch() != tt.expected {
				t.Errorf("expected CanSearch() = %v", tt.expected)
			}
		})
	}
}

func TestRoleConstants(t *testing.T) {
	if RoleAdmin != "admin" {
		t.Errorf("expected RoleAdmin = 'admin', got %s", RoleAdmin)
	}
	if RoleMember != "member" {
		t.Errorf("expected RoleMember = 'member', got %s", RoleMember)
	}
	if RoleViewer != "viewer" {
		t.Errorf("expected RoleViewer = 'viewer', got %s", RoleViewer)
	}
}
