package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
)

type NotificationService interface {
	CreateNotification(ctx context.Context, req CreateNotificationRequest) (notification.Notification, error)
	GetNotificationByID(ctx context.Context, id string) (notification.Notification, error)
	ListNotifications(ctx context.Context, filter notification.Filter, sort pagination.Sort, page pagination.Page) ([]notification.Notification, int, error)
	MarkAsRead(ctx context.Context, id string) (notification.Notification, error)
	GetUnreadCount(ctx context.Context, userID string) (int, error)
	GetUnreadCountsAll(ctx context.Context) ([]notification.UserUnreadCount, error)
	GetPreferences(ctx context.Context, userID string) ([]notification.Preference, error)
	UpdatePreference(ctx context.Context, pref notification.Preference) (notification.Preference, error)
	BulkCreate(ctx context.Context, req BulkNotificationRequest) (BulkResult, error)
}

type NotificationController struct {
	svc NotificationService
	log *zap.Logger
}

func NewNotificationController(svc NotificationService, log *zap.Logger) *NotificationController {
	return &NotificationController{svc: svc, log: log}
}

var allowedSortFields = map[string]bool{
	"created_at": true,
	"type":       true,
	"status":     true,
}

func (c *NotificationController) Create(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r.Context())
	if role != "admin" {
		httpresponse.Forbidden(w, "forbidden", "only admin can create notifications")
		return
	}

	var req CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.UserID == "" {
		httpresponse.BadRequest(w, "validation_error", "user_id is required")
		return
	}
	if req.Title == "" {
		httpresponse.BadRequest(w, "validation_error", "title is required")
		return
	}
	if req.Message == "" {
		httpresponse.BadRequest(w, "validation_error", "message is required")
		return
	}
	if req.Type == "" {
		httpresponse.BadRequest(w, "validation_error", "type is required")
		return
	}

	created, err := c.svc.CreateNotification(r.Context(), req)
	if err != nil {
		c.log.Error("Failed to create notification", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.Created(w, created)
}

func (c *NotificationController) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httpresponse.Unauthorized(w, "unauthorized", "user id not found")
		return
	}

	filter := parseFilter(r, userID)
	sort := pagination.SortFromRequest(r, allowedSortFields, "created_at")
	page := pagination.PageFromRequest(r)

	notifications, total, err := c.svc.ListNotifications(r.Context(), filter, sort, page)
	if err != nil {
		c.log.Error("Failed to list notifications", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if notifications == nil {
		notifications = []notification.Notification{}
	}

	httpresponse.List(w, notifications, total, page.Limit, page.Offset)
}

func (c *NotificationController) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "notification id is required")
		return
	}

	callerID := middleware.GetUserID(r.Context())
	callerRole := middleware.GetUserRole(r.Context())

	existing, err := c.svc.GetNotificationByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, notification.ErrNotificationNotFound) {
			httpresponse.NotFound(w, "notification_not_found", "notification not found")
			return
		}
		c.log.Error("Failed to load notification", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if existing.UserID != callerID && callerRole != "admin" {
		httpresponse.Forbidden(w, "forbidden", "cannot mark another user's notification as read")
		return
	}

	if existing.Status == "read" {
		httpresponse.OK(w, existing)
		return
	}

	updated, err := c.svc.MarkAsRead(r.Context(), id)
	if err != nil {
		if errors.Is(err, notification.ErrNotificationNotFound) {
			httpresponse.NotFound(w, "notification_not_found", "notification not found")
			return
		}
		c.log.Error("Failed to mark notification as read", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, updated)
}

func (c *NotificationController) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httpresponse.Unauthorized(w, "unauthorized", "user id not found")
		return
	}

	count, err := c.svc.GetUnreadCount(r.Context(), userID)
	if err != nil {
		c.log.Error("Failed to get unread count", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, map[string]int{"unread_count": count})
}

func (c *NotificationController) AdminUnreadCounts(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r.Context())
	if role != "admin" {
		httpresponse.Forbidden(w, "forbidden", "only admin can read aggregated unread counts")
		return
	}

	counts, err := c.svc.GetUnreadCountsAll(r.Context())
	if err != nil {
		c.log.Error("Failed to get aggregated unread counts", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if counts == nil {
		counts = []notification.UserUnreadCount{}
	}

	httpresponse.OK(w, counts)
}

func (c *NotificationController) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httpresponse.Unauthorized(w, "unauthorized", "user id not found")
		return
	}

	prefs, err := c.svc.GetPreferences(r.Context(), userID)
	if err != nil {
		c.log.Error("Failed to get preferences", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if prefs == nil {
		prefs = []notification.Preference{}
	}

	httpresponse.OK(w, prefs)
}

func (c *NotificationController) UpdatePreference(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httpresponse.Unauthorized(w, "unauthorized", "user id not found")
		return
	}

	var req struct {
		Type  notification.Type `json:"type"`
		InApp *bool             `json:"in_app"`
		Email *bool             `json:"email"`
		SMS   *bool             `json:"sms"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Type == "" {
		httpresponse.BadRequest(w, "validation_error", "type is required")
		return
	}

	pref := notification.Preference{
		UserID: userID,
		Type:   req.Type,
		InApp:  true,
		Email:  true,
		SMS:    false,
	}
	if req.InApp != nil {
		pref.InApp = *req.InApp
	}
	if req.Email != nil {
		pref.Email = *req.Email
	}
	if req.SMS != nil {
		pref.SMS = *req.SMS
	}

	updated, err := c.svc.UpdatePreference(r.Context(), pref)
	if err != nil {
		c.log.Error("Failed to update preference", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, updated)
}

func (c *NotificationController) Bulk(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r.Context())
	if role != "admin" {
		httpresponse.Forbidden(w, "forbidden", "only admin can send bulk notifications")
		return
	}

	var req BulkNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if len(req.UserIDs) == 0 {
		httpresponse.BadRequest(w, "validation_error", "user_ids is required")
		return
	}
	if req.Title == "" {
		httpresponse.BadRequest(w, "validation_error", "title is required")
		return
	}
	if req.Message == "" {
		httpresponse.BadRequest(w, "validation_error", "message is required")
		return
	}
	if req.Type == "" {
		httpresponse.BadRequest(w, "validation_error", "type is required")
		return
	}

	result, err := c.svc.BulkCreate(r.Context(), req)
	if err != nil {
		c.log.Error("Failed to bulk create notifications", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, result)
}

func parseFilter(r *http.Request, userID string) notification.Filter {
	filter := notification.Filter{
		UserID: &userID,
	}

	if s := r.URL.Query().Get("type"); s != "" {
		t := notification.Type(s)
		filter.Type = &t
	}
	if s := r.URL.Query().Get("status"); s != "" {
		st := notification.Status(s)
		filter.Status = &st
	}
	if s := r.URL.Query().Get("is_read"); s != "" {
		switch s {
		case "true", "1":
			t := true
			filter.IsRead = &t
		case "false", "0":
			f := false
			filter.IsRead = &f
		}
	}

	return filter
}
