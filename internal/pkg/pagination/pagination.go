package pagination

import (
	"net/http"
	"strconv"
)

const (
	defaultLimit = 20
	maxLimit     = 1000
)

type Page struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type Sort struct {
	Field string `json:"field"`
	DESC  bool   `json:"desc"`
}

func (s Sort) Direction() string {
	if s.DESC {
		return "DESC"
	}
	return "ASC"
}

func PageFromRequest(r *http.Request) Page {
	limit := parseIntParam(r, "limit", defaultLimit)
	if limit <= 0 {
		limit = defaultLimit
	} else if limit > maxLimit {
		limit = maxLimit
	}

	offset := parseIntParam(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}

	return Page{Limit: limit, Offset: offset}
}

func SortFromRequest(r *http.Request, allowedFields map[string]bool, defaultField string) Sort {
	field := r.URL.Query().Get("sort_by")
	if field == "" || !allowedFields[field] {
		field = defaultField
	}

	order := r.URL.Query().Get("sort_order")
	desc := order == "desc"

	return Sort{Field: field, DESC: desc}
}

func parseIntParam(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return val
}
