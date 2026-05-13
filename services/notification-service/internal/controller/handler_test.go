package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
)

func withCtx(r *http.Request, userID, role string) *http.Request {
	return r.WithContext(middleware.WithUserContext(r.Context(), userID, role))
}

func setupNotificationController() (*NotificationController, *Service) {
	svc, _, _, _, _ := newTestService()
	log := zap.NewNop()
	ctrl := NewNotificationController(svc, log)
	return ctrl, svc
}

func TestNotificationController_Create_Success(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeOrderCreated,
		Title:   "New Order",
		Message: "Order #123 created",
	})
	req := withCtx(httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body)), "", "admin")
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %+v", resp.Error)
	}
}

func TestNotificationController_Create_Forbidden(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeSystem,
		Title:   "Test",
		Message: "msg",
	})
	req := withCtx(httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body)), "", "operator")
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestNotificationController_Create_MissingUserID(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(CreateNotificationRequest{
		Type:    notification.TypeSystem,
		Title:   "Test",
		Message: "msg",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req = withCtx(req, "", "admin")
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestNotificationController_Create_MissingTitle(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeSystem,
		Message: "msg",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req = withCtx(req, "", "admin")
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestNotificationController_Create_MissingMessage(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(CreateNotificationRequest{
		UserID: "user-1",
		Type:   notification.TypeSystem,
		Title:  "Test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req = withCtx(req, "", "admin")
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestNotificationController_Create_MissingType(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(CreateNotificationRequest{
		UserID:  "user-1",
		Title:   "Test",
		Message: "msg",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req = withCtx(req, "", "admin")
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestNotificationController_List_Success(t *testing.T) {
	ctrl, svc := setupNotificationController()

	_, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID: "user-1", Type: notification.TypeSystem, Title: "Test", Message: "msg",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=20&offset=0", nil)
	req = withCtx(req, "user-1", "")
	rec := httptest.NewRecorder()

	ctrl.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Meta == nil {
		t.Error("expected meta in response")
	}
}

func TestNotificationController_List_NoUserID(t *testing.T) {
	ctrl, _ := setupNotificationController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	rec := httptest.NewRecorder()

	ctrl.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestNotificationController_List_WithTypeFilter(t *testing.T) {
	ctrl, _ := setupNotificationController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?type=system", nil)
	req = withCtx(req, "user-1", "")
	rec := httptest.NewRecorder()

	ctrl.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestNotificationController_MarkAsRead_Success(t *testing.T) {
	ctrl, svc := setupNotificationController()

	created, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID: "user-1", Type: notification.TypeSystem, Title: "Test", Message: "msg",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/"+created.ID+"/read", nil)
	req.SetPathValue("id", created.ID)
	req = req.WithContext(middleware.WithUserContext(req.Context(), "user-1", "operator"))
	rec := httptest.NewRecorder()

	ctrl.MarkAsRead(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNotificationController_MarkAsRead_NotFound(t *testing.T) {
	ctrl, _ := setupNotificationController()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/nonexistent/read", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.MarkAsRead(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestNotificationController_UnreadCount_Success(t *testing.T) {
	ctrl, svc := setupNotificationController()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
			UserID: "user-1", Type: notification.TypeSystem, Title: "Test", Message: "msg",
		})
		if err != nil {
			t.Fatalf("CreateNotification failed: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	req = withCtx(req, "user-1", "")
	rec := httptest.NewRecorder()

	ctrl.UnreadCount(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %+v", resp.Error)
	}
}

func TestNotificationController_UnreadCount_NoUserID(t *testing.T) {
	ctrl, _ := setupNotificationController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	rec := httptest.NewRecorder()

	ctrl.UnreadCount(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestNotificationController_GetPreferences_Success(t *testing.T) {
	ctrl, _ := setupNotificationController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/preferences", nil)
	req = withCtx(req, "user-1", "")
	rec := httptest.NewRecorder()

	ctrl.GetPreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNotificationController_GetPreferences_NoUserID(t *testing.T) {
	ctrl, _ := setupNotificationController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/preferences", nil)
	rec := httptest.NewRecorder()

	ctrl.GetPreferences(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestNotificationController_UpdatePreference_Success(t *testing.T) {
	ctrl, _ := setupNotificationController()

	inApp := false
	email := true
	sms := true
	body, _ := json.Marshal(map[string]any{
		"type":   "low_stock",
		"in_app": inApp,
		"email":  email,
		"sms":    sms,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(body))
	req = withCtx(req, "user-1", "")
	rec := httptest.NewRecorder()

	ctrl.UpdatePreference(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNotificationController_UpdatePreference_NoUserID(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(map[string]any{"type": "system"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.UpdatePreference(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestNotificationController_UpdatePreference_MissingType(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(map[string]any{"in_app": false})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(body))
	req = withCtx(req, "user-1", "")
	rec := httptest.NewRecorder()

	ctrl.UpdatePreference(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestNotificationController_Bulk_Success(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(BulkNotificationRequest{
		UserIDs: []string{"user-1", "user-2"},
		Type:    notification.TypeSystem,
		Title:   "Announcement",
		Message: "System maintenance",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/bulk", bytes.NewReader(body))
	req = withCtx(req, "", "admin")
	rec := httptest.NewRecorder()

	ctrl.Bulk(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNotificationController_Bulk_Forbidden(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(BulkNotificationRequest{
		UserIDs: []string{"user-1"},
		Type:    notification.TypeSystem,
		Title:   "Test",
		Message: "msg",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/bulk", bytes.NewReader(body))
	req = withCtx(req, "", "operator")
	rec := httptest.NewRecorder()

	ctrl.Bulk(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestNotificationController_Bulk_MissingUserIDs(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(BulkNotificationRequest{
		Type:    notification.TypeSystem,
		Title:   "Test",
		Message: "msg",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/bulk", bytes.NewReader(body))
	req = withCtx(req, "", "admin")
	rec := httptest.NewRecorder()

	ctrl.Bulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestNotificationController_Bulk_MissingTitle(t *testing.T) {
	ctrl, _ := setupNotificationController()

	body, _ := json.Marshal(BulkNotificationRequest{
		UserIDs: []string{"user-1"},
		Type:    notification.TypeSystem,
		Message: "msg",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/bulk", bytes.NewReader(body))
	req = withCtx(req, "", "admin")
	rec := httptest.NewRecorder()

	ctrl.Bulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
