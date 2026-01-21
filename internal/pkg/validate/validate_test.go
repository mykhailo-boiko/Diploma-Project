package validate

import (
	"errors"
	"testing"
)

type testStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=0,lte=150"`
}

func TestValidator_Struct_Valid(t *testing.T) {
	v := New()
	s := testStruct{Name: "John", Email: "john@example.com", Age: 30}

	err := v.Struct(s)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidator_Struct_MissingRequired(t *testing.T) {
	v := New()
	s := testStruct{Email: "john@example.com", Age: 30}

	err := v.Struct(s)
	if err == nil {
		t.Fatal("expected validation error for missing Name")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	fields := ve.Fields()
	if _, ok := fields["Name"]; !ok {
		t.Error("expected 'Name' in validation errors")
	}
}

func TestValidator_Struct_InvalidEmail(t *testing.T) {
	v := New()
	s := testStruct{Name: "John", Email: "not-an-email", Age: 30}

	err := v.Struct(s)
	if err == nil {
		t.Fatal("expected validation error for invalid email")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	fields := ve.Fields()
	if _, ok := fields["Email"]; !ok {
		t.Error("expected 'Email' in validation errors")
	}
}

func TestValidator_Struct_MultipleErrors(t *testing.T) {
	v := New()
	s := testStruct{Age: -1}

	err := v.Struct(s)
	if err == nil {
		t.Fatal("expected validation errors")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	if len(ve.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d", len(ve.Errors))
	}

	errMsg := ve.Error()
	if errMsg == "" {
		t.Error("expected non-empty error message")
	}
}
