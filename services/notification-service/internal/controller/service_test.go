package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
)

type mockStorage struct {
	mu            sync.Mutex
	notifications map[string]notification.Notification
	preferences   map[string]notification.Preference
	nextID        int
	nextPrefID    int
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		notifications: make(map[string]notification.Notification),
		preferences:   make(map[string]notification.Preference),
	}
}

func (m *mockStorage) CreateNotification(_ context.Context, n notification.Notification) (notification.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	n.ID = fmt.Sprintf("notif-%d", m.nextID)
	n.Status = notification.StatusPending
	n.CreatedAt = time.Now().UTC()
	m.notifications[n.ID] = n
	return n, nil
}

func (m *mockStorage) GetNotificationByID(_ context.Context, id string) (notification.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	n, ok := m.notifications[id]
	if !ok {
		return notification.Notification{}, notification.ErrNotificationNotFound
	}
	return n, nil
}

func (m *mockStorage) ListNotifications(_ context.Context, filter notification.Filter, _ pagination.Sort, _ pagination.Page) ([]notification.Notification, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []notification.Notification
	for _, n := range m.notifications {
		if filter.UserID != nil && n.UserID != *filter.UserID {
			continue
		}
		if filter.Type != nil && n.Type != *filter.Type {
			continue
		}
		if filter.Status != nil && n.Status != *filter.Status {
			continue
		}
		result = append(result, n)
	}
	return result, len(result), nil
}

func (m *mockStorage) MarkAsRead(_ context.Context, id string) (notification.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	n, ok := m.notifications[id]
	if !ok {
		return notification.Notification{}, notification.ErrNotificationNotFound
	}
	if n.Status != notification.StatusPending {
		return notification.Notification{}, notification.ErrNotificationNotFound
	}
	n.Status = notification.StatusRead
	now := time.Now().UTC()
	n.ReadAt = &now
	m.notifications[id] = n
	return n, nil
}

func (m *mockStorage) GetUnreadCount(_ context.Context, userID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int
	for _, n := range m.notifications {
		if n.UserID == userID && n.Status == notification.StatusPending {
			count++
		}
	}
	return count, nil
}

func (m *mockStorage) GetUnreadCountsAll(_ context.Context) ([]notification.UserUnreadCount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	totals := make(map[string]int)
	for _, n := range m.notifications {
		if n.Status == notification.StatusPending {
			totals[n.UserID]++
		}
	}

	result := make([]notification.UserUnreadCount, 0, len(totals))
	for uid, c := range totals {
		result = append(result, notification.UserUnreadCount{UserID: uid, UnreadCount: c})
	}
	return result, nil
}

func (m *mockStorage) GetPreferences(_ context.Context, userID string) ([]notification.Preference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []notification.Preference
	for _, p := range m.preferences {
		if p.UserID == userID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockStorage) UpsertPreference(_ context.Context, pref notification.Preference) (notification.Preference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := pref.UserID + string(pref.Type)
	existing, ok := m.preferences[key]
	if ok {
		existing.InApp = pref.InApp
		existing.Email = pref.Email
		existing.SMS = pref.SMS
		existing.UpdatedAt = time.Now().UTC()
		m.preferences[key] = existing
		return existing, nil
	}

	m.nextPrefID++
	pref.ID = fmt.Sprintf("pref-%d", m.nextPrefID)
	pref.UpdatedAt = time.Now().UTC()
	m.preferences[key] = pref
	return pref, nil
}

type mockWSPusher struct {
	mu     sync.Mutex
	pushed []notification.Notification
}

func (m *mockWSPusher) Push(_ string, n notification.Notification) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pushed = append(m.pushed, n)
}

type mockDelivery struct {
	mu   sync.Mutex
	sent []notification.Notification
}

func (m *mockDelivery) Send(_ context.Context, n notification.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, n)
	return nil
}

func newTestService() (*Service, *mockStorage, *mockWSPusher, *mockDelivery, *mockDelivery) {
	storage := newMockStorage()
	wsPusher := &mockWSPusher{}
	email := &mockDelivery{}
	sms := &mockDelivery{}
	return NewService(storage, email, sms, wsPusher, nil, zap.NewNop()), storage, wsPusher, email, sms
}

func TestCreateNotification(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	created, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeOrderCreated,
		Title:   "New Order",
		Message: "Order #123 has been created",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	if created.UserID != "user-1" {
		t.Errorf("expected user_id 'user-1', got %q", created.UserID)
	}
	if created.Status != notification.StatusPending {
		t.Errorf("expected status pending, got %s", created.Status)
	}
	if created.Type != notification.TypeOrderCreated {
		t.Errorf("expected type order_created, got %s", created.Type)
	}
	if created.Title != "New Order" {
		t.Errorf("expected title 'New Order', got %q", created.Title)
	}
}

func TestCreateNotification_WebSocketPush(t *testing.T) {
	svc, _, wsPusher, _, _ := newTestService()

	created, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeOrderCreated,
		Title:   "New Order",
		Message: "Order #123 created",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	wsPusher.mu.Lock()
	defer wsPusher.mu.Unlock()

	if len(wsPusher.pushed) != 1 {
		t.Fatalf("expected 1 WebSocket push, got %d", len(wsPusher.pushed))
	}
	if wsPusher.pushed[0].ID != created.ID {
		t.Errorf("expected push for notification %s, got %s", created.ID, wsPusher.pushed[0].ID)
	}
}

func TestCreateNotification_EmailDelivery(t *testing.T) {
	svc, _, _, email, _ := newTestService()

	_, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeSystem,
		Title:   "Test",
		Message: "Test message",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	email.mu.Lock()
	defer email.mu.Unlock()

	if len(email.sent) != 1 {
		t.Errorf("expected 1 email sent, got %d", len(email.sent))
	}
}

func TestCreateNotification_PreferenceDisablesInApp(t *testing.T) {
	svc, storage, wsPusher, _, _ := newTestService()

	_, err := storage.UpsertPreference(t.Context(), notification.Preference{
		UserID: "user-1",
		Type:   notification.TypeLowStock,
		InApp:  false,
		Email:  true,
		SMS:    false,
	})
	if err != nil {
		t.Fatalf("UpsertPreference failed: %v", err)
	}

	created, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeLowStock,
		Title:   "Low Stock",
		Message: "Product XYZ is low",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	if created.ID != "" {
		t.Errorf("expected empty notification (in_app disabled), got ID %s", created.ID)
	}

	wsPusher.mu.Lock()
	defer wsPusher.mu.Unlock()
	if len(wsPusher.pushed) != 0 {
		t.Errorf("expected 0 WebSocket pushes, got %d", len(wsPusher.pushed))
	}
}

func TestGetNotificationByID(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	created, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeSystem,
		Title:   "Test",
		Message: "Test message",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	fetched, err := svc.GetNotificationByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("GetNotificationByID failed: %v", err)
	}

	if fetched.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, fetched.ID)
	}
}

func TestGetNotificationByID_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	_, err := svc.GetNotificationByID(t.Context(), "nonexistent")
	if !errors.Is(err, notification.ErrNotificationNotFound) {
		t.Errorf("expected ErrNotificationNotFound, got %v", err)
	}
}

func TestListNotifications(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
			UserID:  "user-1",
			Type:    notification.TypeSystem,
			Title:   fmt.Sprintf("Notification %d", i),
			Message: "Test message",
		})
		if err != nil {
			t.Fatalf("CreateNotification failed: %v", err)
		}
	}

	_, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID:  "user-2",
		Type:    notification.TypeSystem,
		Title:   "Other User",
		Message: "Test",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	userID := "user-1"
	filter := notification.Filter{UserID: &userID}
	notifications, total, err := svc.ListNotifications(t.Context(), filter, pagination.Sort{Field: "created_at"}, pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(notifications) != 3 {
		t.Errorf("expected 3 notifications, got %d", len(notifications))
	}
}

func TestListNotifications_WithTypeFilter(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	_, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID: "user-1", Type: notification.TypeOrderCreated, Title: "Order", Message: "msg",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	_, err = svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID: "user-1", Type: notification.TypeLowStock, Title: "Stock", Message: "msg",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	userID := "user-1"
	nType := notification.TypeOrderCreated
	filter := notification.Filter{UserID: &userID, Type: &nType}
	notifications, total, err := svc.ListNotifications(t.Context(), filter, pagination.Sort{Field: "created_at"}, pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}

	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(notifications) != 1 {
		t.Errorf("expected 1 notification, got %d", len(notifications))
	}
}

func TestMarkAsRead(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	created, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeSystem,
		Title:   "Test",
		Message: "Test message",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	updated, err := svc.MarkAsRead(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	if updated.Status != notification.StatusRead {
		t.Errorf("expected status read, got %s", updated.Status)
	}
}

func TestMarkAsRead_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	_, err := svc.MarkAsRead(t.Context(), "nonexistent")
	if !errors.Is(err, notification.ErrNotificationNotFound) {
		t.Errorf("expected ErrNotificationNotFound, got %v", err)
	}
}

func TestMarkAsRead_AlreadyRead(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	created, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeSystem,
		Title:   "Test",
		Message: "Test message",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	_, err = svc.MarkAsRead(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	_, err = svc.MarkAsRead(t.Context(), created.ID)
	if !errors.Is(err, notification.ErrNotificationNotFound) {
		t.Errorf("expected ErrNotificationNotFound for already-read notification, got %v", err)
	}
}

func TestGetUnreadCount(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateNotification(t.Context(), CreateNotificationRequest{
			UserID:  "user-1",
			Type:    notification.TypeSystem,
			Title:   fmt.Sprintf("Notif %d", i),
			Message: "msg",
		})
		if err != nil {
			t.Fatalf("CreateNotification failed: %v", err)
		}
	}

	count, err := svc.GetUnreadCount(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 unread, got %d", count)
	}

	userID := "user-1"
	notifications, _, err := svc.ListNotifications(t.Context(), notification.Filter{UserID: &userID}, pagination.Sort{Field: "created_at"}, pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}

	_, err = svc.MarkAsRead(t.Context(), notifications[0].ID)
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	count, err = svc.GetUnreadCount(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 unread after reading one, got %d", count)
	}
}

func TestGetUnreadCount_NoNotifications(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	count, err := svc.GetUnreadCount(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 unread, got %d", count)
	}
}

func TestGetPreferences_Empty(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	prefs, err := svc.GetPreferences(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("GetPreferences failed: %v", err)
	}

	if len(prefs) != 0 {
		t.Errorf("expected 0 preferences, got %d", len(prefs))
	}
}

func TestUpdatePreference(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	updated, err := svc.UpdatePreference(t.Context(), notification.Preference{
		UserID: "user-1",
		Type:   notification.TypeLowStock,
		InApp:  false,
		Email:  true,
		SMS:    true,
	})
	if err != nil {
		t.Fatalf("UpdatePreference failed: %v", err)
	}

	if updated.UserID != "user-1" {
		t.Errorf("expected user_id 'user-1', got %q", updated.UserID)
	}
	if updated.InApp {
		t.Error("expected in_app false")
	}
	if !updated.Email {
		t.Error("expected email true")
	}
	if !updated.SMS {
		t.Error("expected sms true")
	}

	prefs, err := svc.GetPreferences(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("GetPreferences failed: %v", err)
	}
	if len(prefs) != 1 {
		t.Fatalf("expected 1 preference, got %d", len(prefs))
	}
	if prefs[0].Type != notification.TypeLowStock {
		t.Errorf("expected type low_stock, got %s", prefs[0].Type)
	}
}

func TestUpdatePreference_Upsert(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	_, err := svc.UpdatePreference(t.Context(), notification.Preference{
		UserID: "user-1",
		Type:   notification.TypeOrderCreated,
		InApp:  true,
		Email:  true,
		SMS:    false,
	})
	if err != nil {
		t.Fatalf("UpdatePreference failed: %v", err)
	}

	updated, err := svc.UpdatePreference(t.Context(), notification.Preference{
		UserID: "user-1",
		Type:   notification.TypeOrderCreated,
		InApp:  true,
		Email:  false,
		SMS:    true,
	})
	if err != nil {
		t.Fatalf("UpdatePreference failed: %v", err)
	}

	if updated.Email {
		t.Error("expected email false after update")
	}
	if !updated.SMS {
		t.Error("expected sms true after update")
	}

	prefs, err := svc.GetPreferences(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("GetPreferences failed: %v", err)
	}
	if len(prefs) != 1 {
		t.Errorf("expected 1 preference after upsert, got %d", len(prefs))
	}
}

func TestBulkCreate(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	result, err := svc.BulkCreate(t.Context(), BulkNotificationRequest{
		UserIDs: []string{"user-1", "user-2", "user-3"},
		Type:    notification.TypeSystem,
		Title:   "Announcement",
		Message: "System maintenance tonight",
	})
	if err != nil {
		t.Fatalf("BulkCreate failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
	if result.Success != 3 {
		t.Errorf("expected 3 success, got %d", result.Success)
	}
	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}

	for _, uid := range []string{"user-1", "user-2", "user-3"} {
		count, err := svc.GetUnreadCount(t.Context(), uid)
		if err != nil {
			t.Fatalf("GetUnreadCount failed for %s: %v", uid, err)
		}
		if count != 1 {
			t.Errorf("expected 1 unread for %s, got %d", uid, count)
		}
	}
}

func TestCreateNotification_SMSPreference(t *testing.T) {
	svc, storage, _, _, sms := newTestService()

	_, err := storage.UpsertPreference(t.Context(), notification.Preference{
		UserID: "user-1",
		Type:   notification.TypeOrderCreated,
		InApp:  true,
		Email:  false,
		SMS:    true,
	})
	if err != nil {
		t.Fatalf("UpsertPreference failed: %v", err)
	}

	_, err = svc.CreateNotification(t.Context(), CreateNotificationRequest{
		UserID:  "user-1",
		Type:    notification.TypeOrderCreated,
		Title:   "New Order",
		Message: "Order created",
	})
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	sms.mu.Lock()
	defer sms.mu.Unlock()

	if len(sms.sent) != 1 {
		t.Errorf("expected 1 SMS sent, got %d", len(sms.sent))
	}
}
