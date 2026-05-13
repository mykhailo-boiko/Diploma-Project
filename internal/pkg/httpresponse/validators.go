package httpresponse

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func ValidateUUIDPath(w http.ResponseWriter, r *http.Request, paramName string) (string, bool) {
	val := r.PathValue(paramName)
	if val == "" {
		MissingField(w, paramName,
			"UUID v4 in URL path",
			"Specify the ID in the URL path.")
		return "", false
	}
	if _, err := uuid.Parse(val); err != nil {
		InvalidField(w, paramName,
			"valid UUID v4", val,
			"Use UUID format (e.g. 9ebcaf0f-c50d-4f36-b417-c3fa7477fc8c).",
			"9ebcaf0f-c50d-4f36-b417-c3fa7477fc8c")
		return "", false
	}
	return val, true
}

func ParseDateFromQuery(r *http.Request, name string) (time.Time, bool, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return time.Time{}, false, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, true, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true, nil
	}
	return time.Time{}, true, &DateParseError{Field: name, Received: raw}
}

type DateParseError struct {
	Field    string
	Received string
}

func (e *DateParseError) Error() string {
	return "invalid date format for field " + e.Field + ": " + e.Received
}

func WriteDateParseError(w http.ResponseWriter, err *DateParseError) {
	InvalidField(w, err.Field,
		"ISO date 'YYYY-MM-DD' or RFC3339 datetime", err.Received,
		"Use ISO date (e.g. 2026-05-13) or RFC3339 (e.g. 2026-05-13T15:00:00Z).",
		"2026-05-13", "2026-05-13T15:00:00Z")
}

func CapLimit(rawLimit string, defaultLimit, maxLimit int) (int, bool) {
	if rawLimit == "" {
		return defaultLimit, false
	}
	n, err := strconv.Atoi(rawLimit)
	if err != nil {
		return defaultLimit, false
	}
	if n < 1 {
		return defaultLimit, false
	}
	if n > maxLimit {
		return maxLimit, true
	}
	return n, false
}
