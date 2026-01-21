package validate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	v *validator.Validate
}

func New() *Validator {
	return &Validator{v: validator.New(validator.WithRequiredStructEnabled())}
}

func (val *Validator) Struct(s any) error {
	err := val.v.Struct(s)
	if err == nil {
		return nil
	}

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return fmt.Errorf("validation failed: %w", err)
	}

	return &ValidationError{Errors: ve}
}

type ValidationError struct {
	Errors validator.ValidationErrors
}

func (e *ValidationError) Error() string {
	var msgs []string
	for _, fe := range e.Errors {
		msgs = append(msgs, fmt.Sprintf("field '%s' failed on '%s' tag", fe.Field(), fe.Tag()))
	}
	return strings.Join(msgs, "; ")
}

func (e *ValidationError) Fields() map[string]string {
	result := make(map[string]string, len(e.Errors))
	for _, fe := range e.Errors {
		result[fe.Field()] = fmt.Sprintf("failed on '%s' validation", fe.Tag())
	}
	return result
}
