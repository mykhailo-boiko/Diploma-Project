package controller

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		give    string
		wantErr error
	}{
		{"too short", "Aa1", ErrPasswordTooShort},
		{"no upper", "abcdef12", ErrPasswordNoUpper},
		{"no lower", "ABCDEF12", ErrPasswordNoLower},
		{"no digit", "Abcdefgh", ErrPasswordNoDigit},
		{"valid", "Pass1234", nil},
		{"valid complex", "StrongPass!1", nil},
		{"empty", "", ErrPasswordTooShort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.give)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error %v, want %v", err, tt.wantErr)
			}
		})
	}
}
