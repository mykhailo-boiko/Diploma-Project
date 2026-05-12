package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageFromRequest_Defaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items", nil)
	page := PageFromRequest(r)

	if page.Limit != defaultLimit {
		t.Errorf("expected limit %d, got %d", defaultLimit, page.Limit)
	}
	if page.Offset != 0 {
		t.Errorf("expected offset 0, got %d", page.Offset)
	}
}

func TestPageFromRequest_Custom(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=50&offset=10", nil)
	page := PageFromRequest(r)

	if page.Limit != 50 {
		t.Errorf("expected limit 50, got %d", page.Limit)
	}
	if page.Offset != 10 {
		t.Errorf("expected offset 10, got %d", page.Offset)
	}
}

func TestPageFromRequest_ExceedsMax(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=5000", nil)
	page := PageFromRequest(r)

	if page.Limit != defaultLimit {
		t.Errorf("expected limit %d for exceeding max (%d), got %d", defaultLimit, maxLimit, page.Limit)
	}
}

func TestPageFromRequest_AllowsLargeLimitUpToMax(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=500", nil)
	page := PageFromRequest(r)
	if page.Limit != 500 {
		t.Errorf("expected limit 500 (below max %d), got %d", maxLimit, page.Limit)
	}
}

func TestPageFromRequest_NegativeOffset(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?offset=-5", nil)
	page := PageFromRequest(r)

	if page.Offset != 0 {
		t.Errorf("expected offset 0 for negative, got %d", page.Offset)
	}
}

func TestPageFromRequest_InvalidValues(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=abc&offset=xyz", nil)
	page := PageFromRequest(r)

	if page.Limit != defaultLimit {
		t.Errorf("expected default limit, got %d", page.Limit)
	}
	if page.Offset != 0 {
		t.Errorf("expected offset 0, got %d", page.Offset)
	}
}

func TestSortFromRequest_Default(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items", nil)
	allowed := map[string]bool{"name": true, "created_at": true}
	sort := SortFromRequest(r, allowed, "created_at")

	if sort.Field != "created_at" {
		t.Errorf("expected field 'created_at', got %q", sort.Field)
	}
	if sort.DESC {
		t.Error("expected ASC by default")
	}
}

func TestSortFromRequest_Custom(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?sort_by=name&sort_order=desc", nil)
	allowed := map[string]bool{"name": true, "created_at": true}
	sort := SortFromRequest(r, allowed, "created_at")

	if sort.Field != "name" {
		t.Errorf("expected field 'name', got %q", sort.Field)
	}
	if !sort.DESC {
		t.Error("expected DESC")
	}
}

func TestSortFromRequest_DisallowedField(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?sort_by=hacked_field", nil)
	allowed := map[string]bool{"name": true}
	sort := SortFromRequest(r, allowed, "name")

	if sort.Field != "name" {
		t.Errorf("expected fallback to 'name', got %q", sort.Field)
	}
}

func TestSortDirection(t *testing.T) {
	asc := Sort{Field: "name", DESC: false}
	if asc.Direction() != "ASC" {
		t.Errorf("expected ASC, got %q", asc.Direction())
	}

	desc := Sort{Field: "name", DESC: true}
	if desc.Direction() != "DESC" {
		t.Errorf("expected DESC, got %q", desc.Direction())
	}
}
