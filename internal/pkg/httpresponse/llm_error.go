package httpresponse

import "net/http"

type LLMError struct {
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	Field      string   `json:"field,omitempty"`
	Expected   string   `json:"expected,omitempty"`
	Received   any      `json:"received,omitempty"`
	Suggestion string   `json:"suggestion,omitempty"`
	Examples   []string `json:"examples,omitempty"`
	DocRef     string   `json:"doc_ref,omitempty"`
}

func (e LLMError) toData() map[string]any {
	out := map[string]any{}
	if e.Field != "" {
		out["field"] = e.Field
	}
	if e.Expected != "" {
		out["expected"] = e.Expected
	}
	if e.Received != nil {
		out["received"] = e.Received
	}
	if e.Suggestion != "" {
		out["suggestion"] = e.Suggestion
	}
	if len(e.Examples) > 0 {
		out["examples"] = e.Examples
	}
	if e.DocRef != "" {
		out["doc_ref"] = e.DocRef
	}
	return out
}

func ValidationError(w http.ResponseWriter, e LLMError) {
	if e.Code == "" {
		e.Code = "validation_error"
	}
	ErrWithData(w, http.StatusBadRequest, e.Code, e.Message, e.toData())
}

func NotFoundError(w http.ResponseWriter, e LLMError) {
	if e.Code == "" {
		e.Code = "not_found"
	}
	ErrWithData(w, http.StatusNotFound, e.Code, e.Message, e.toData())
}

func ConflictError(w http.ResponseWriter, e LLMError) {
	if e.Code == "" {
		e.Code = "conflict"
	}
	ErrWithData(w, http.StatusConflict, e.Code, e.Message, e.toData())
}

func ForbiddenError(w http.ResponseWriter, e LLMError) {
	if e.Code == "" {
		e.Code = "forbidden"
	}
	ErrWithData(w, http.StatusForbidden, e.Code, e.Message, e.toData())
}

func RateLimitError(w http.ResponseWriter, e LLMError) {
	if e.Code == "" {
		e.Code = "rate_limit_exceeded"
	}
	ErrWithData(w, http.StatusTooManyRequests, e.Code, e.Message, e.toData())
}

func MissingField(w http.ResponseWriter, field, expected, suggestion string, examples ...string) {
	ValidationError(w, LLMError{
		Code:       "missing_field",
		Message:    "required field '" + field + "' is missing",
		Field:      field,
		Expected:   expected,
		Suggestion: suggestion,
		Examples:   examples,
	})
}

func InvalidField(w http.ResponseWriter, field, expected string, received any, suggestion string, examples ...string) {
	ValidationError(w, LLMError{
		Code:       "invalid_field",
		Message:    "field '" + field + "' has invalid value",
		Field:      field,
		Expected:   expected,
		Received:   received,
		Suggestion: suggestion,
		Examples:   examples,
	})
}

func InvalidBody(w http.ResponseWriter, parseErr string) {
	ValidationError(w, LLMError{
		Code:       "invalid_body",
		Message:    "request body could not be parsed as JSON",
		Expected:   "valid JSON object matching the endpoint schema",
		Received:   parseErr,
		Suggestion: "Check that the request body is well-formed JSON and that field types match the documented schema",
	})
}

func InvalidTransition(w http.ResponseWriter, entity, from, to string, allowed []string) {
	suggestion := "Check the documented status transitions for this entity"
	if len(allowed) > 0 {
		suggestion = "From '" + from + "', allowed next statuses are: " + joinQuoted(allowed)
	}
	ConflictError(w, LLMError{
		Code:       "invalid_status_transition",
		Message:    entity + " cannot transition from '" + from + "' to '" + to + "'",
		Field:      "status",
		Expected:   "one of allowed next statuses",
		Received:   to,
		Suggestion: suggestion,
		Examples:   allowed,
	})
}

func joinQuoted(items []string) string {
	out := ""
	for i, it := range items {
		if i > 0 {
			out += ", "
		}
		out += "'" + it + "'"
	}
	return out
}
