package httpresponse

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestMissingField_StructuredBody(t *testing.T) {
	w := httptest.NewRecorder()
	MissingField(w, "customer_name", "non-empty string", "Provide customer name", "Yuliia Morozenko")

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	body := decode(t, w)
	err, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %v", body)
	}
	if err["code"] != "missing_field" {
		t.Errorf("code = %v, want missing_field", err["code"])
	}
	data, _ := err["data"].(map[string]any)
	if data["field"] != "customer_name" {
		t.Errorf("data.field = %v", data["field"])
	}
	if data["expected"] != "non-empty string" {
		t.Errorf("data.expected = %v", data["expected"])
	}
	if data["suggestion"] == nil {
		t.Errorf("suggestion missing")
	}
	examples, _ := data["examples"].([]any)
	if len(examples) != 1 || examples[0] != "Yuliia Morozenko" {
		t.Errorf("examples = %v", examples)
	}
}

func TestInvalidField_RecordsReceived(t *testing.T) {
	w := httptest.NewRecorder()
	InvalidField(w, "quantity", "positive int", -5, "use positive value", "1", "5")
	body := decode(t, w)
	err := body["error"].(map[string]any)
	data := err["data"].(map[string]any)
	if data["received"].(float64) != -5 {
		t.Errorf("received = %v", data["received"])
	}
	if data["field"] != "quantity" {
		t.Errorf("field = %v", data["field"])
	}
}

func TestInvalidTransition_ListsAllowed(t *testing.T) {
	w := httptest.NewRecorder()
	InvalidTransition(w, "order", "pending", "delivered", []string{"confirmed", "cancelled"})
	if w.Code != 409 {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	body := decode(t, w)
	err := body["error"].(map[string]any)
	if err["code"] != "invalid_status_transition" {
		t.Errorf("code = %v", err["code"])
	}
	data := err["data"].(map[string]any)
	suggestion, _ := data["suggestion"].(string)
	if suggestion == "" {
		t.Errorf("suggestion missing")
	}
}

func TestInvalidBody_ReturnsValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	InvalidBody(w, "unexpected EOF")
	if w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
	body := decode(t, w)
	err := body["error"].(map[string]any)
	if err["code"] != "invalid_body" {
		t.Errorf("code = %v", err["code"])
	}
}

func TestRateLimitError_Sets429(t *testing.T) {
	w := httptest.NewRecorder()
	RateLimitError(w, LLMError{Message: "too fast"})
	if w.Code != 429 {
		t.Fatalf("status = %d, want 429", w.Code)
	}
}
