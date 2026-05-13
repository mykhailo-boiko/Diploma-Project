package controller

import (
	"errors"
	"unicode"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
)

var (
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong   = errors.New("password must be at most 128 characters")
	ErrPasswordNoUpper   = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower   = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit   = errors.New("password must contain at least one digit")
)

func ValidatePassword(pwd string) error {
	if len(pwd) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(pwd) > MaxPasswordLength {
		return ErrPasswordTooLong
	}
	var hasUpper, hasLower, hasDigit bool
	for _, c := range pwd {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}
	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasDigit {
		return ErrPasswordNoDigit
	}
	return nil
}
