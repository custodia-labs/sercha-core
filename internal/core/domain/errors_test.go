package domain

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrNotFound", ErrNotFound, "not found"},
		{"ErrAlreadyExists", ErrAlreadyExists, "already exists"},
		{"ErrInvalidInput", ErrInvalidInput, "invalid input"},
		{"ErrUnauthorized", ErrUnauthorized, "unauthorized"},
		{"ErrForbidden", ErrForbidden, "forbidden"},
		{"ErrSyncInProgress", ErrSyncInProgress, "sync already in progress"},
		{"ErrConnectorNotFound", ErrConnectorNotFound, "connector not found"},
		{"ErrTokenExpired", ErrTokenExpired, "token expired"},
		{"ErrTokenInvalid", ErrTokenInvalid, "token invalid"},
		{"ErrSessionNotFound", ErrSessionNotFound, "session not found"},
		{"ErrInvalidCredentials", ErrInvalidCredentials, "invalid credentials"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("expected %q, got %q", tt.msg, tt.err.Error())
			}
		})
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	allErrors := []error{
		ErrNotFound,
		ErrAlreadyExists,
		ErrInvalidInput,
		ErrUnauthorized,
		ErrForbidden,
		ErrSyncInProgress,
		ErrConnectorNotFound,
		ErrTokenExpired,
		ErrTokenInvalid,
		ErrSessionNotFound,
		ErrInvalidCredentials,
	}

	for i, err1 := range allErrors {
		for j, err2 := range allErrors {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("errors should be distinct: %v and %v", err1, err2)
			}
		}
	}
}

func TestErrorsIs(t *testing.T) {
	// Test that errors.Is works correctly
	if !errors.Is(ErrNotFound, ErrNotFound) {
		t.Error("ErrNotFound should match itself")
	}

	if errors.Is(ErrNotFound, ErrUnauthorized) {
		t.Error("ErrNotFound should not match ErrUnauthorized")
	}
}
