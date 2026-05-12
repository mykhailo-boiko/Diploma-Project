package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserContext_PropagatesAllHeaders(t *testing.T) {
	var (
		gotID    string
		gotRole  string
		gotEmail string
		gotTrace string
	)
	h := UserContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = GetUserID(r.Context())
		gotRole = GetUserRole(r.Context())
		gotEmail = GetUserEmail(r.Context())
		gotTrace = GetTraceID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-User-ID", "user-42")
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Email", "a@b.c")
	req.Header.Set("X-Trace-ID", "trace-abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if gotID != "user-42" {
		t.Errorf("user_id = %q", gotID)
	}
	if gotRole != "admin" {
		t.Errorf("role = %q", gotRole)
	}
	if gotEmail != "a@b.c" {
		t.Errorf("email = %q", gotEmail)
	}
	if gotTrace != "trace-abc" {
		t.Errorf("trace_id = %q", gotTrace)
	}
}

func TestUserContext_AbsentHeadersReturnEmptyStrings(t *testing.T) {
	h := UserContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUserID(r.Context()) != "" {
			t.Errorf("user_id should be empty")
		}
		if GetTraceID(r.Context()) != "" {
			t.Errorf("trace_id should be empty")
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
}
