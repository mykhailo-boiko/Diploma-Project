package httpresponse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOK(t *testing.T) {
	w := httptest.NewRecorder()
	OK(w, map[string]string{"name": "test"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Error("expected no error in response")
	}
	if resp.Data == nil {
		t.Error("expected data in response")
	}
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	Created(w, map[string]string{"id": "123"})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestList(t *testing.T) {
	w := httptest.NewRecorder()
	List(w, []string{"a", "b"}, 100, 20, 0)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Meta == nil {
		t.Fatal("expected meta in response")
	}
	if resp.Meta.Total != 100 {
		t.Errorf("expected total 100, got %d", resp.Meta.Total)
	}
	if resp.Meta.Limit != 20 {
		t.Errorf("expected limit 20, got %d", resp.Meta.Limit)
	}
	if resp.Meta.Offset != 0 {
		t.Errorf("expected offset 0, got %d", resp.Meta.Offset)
	}
}

func TestErr(t *testing.T) {
	w := httptest.NewRecorder()
	Err(w, http.StatusBadRequest, "validation_error", "invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != "validation_error" {
		t.Errorf("expected code 'validation_error', got %q", resp.Error.Code)
	}
	if resp.Error.Message != "invalid input" {
		t.Errorf("expected message 'invalid input', got %q", resp.Error.Message)
	}
	if resp.Data != nil {
		t.Error("expected no data in error response")
	}
}

func TestErrWithData(t *testing.T) {
	w := httptest.NewRecorder()
	ErrWithData(w, http.StatusNotFound, "order_not_found", "order not found", map[string]any{"order_id": "abc"})

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Data["order_id"] != "abc" {
		t.Errorf("expected order_id 'abc', got %v", resp.Error.Data["order_id"])
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "not_found", "resource not found")
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "bad_request", "bad request")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Forbidden(w, "forbidden", "access denied")
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Unauthorized(w, "unauthorized", "not authenticated")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalError(w, "internal_error", "something went wrong")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestContentType(t *testing.T) {
	w := httptest.NewRecorder()
	OK(w, nil)
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}
