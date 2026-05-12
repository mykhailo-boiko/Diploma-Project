package controller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
)

type DeliveryAdapter interface {
	Send(ctx context.Context, n notification.Notification) error
}

type WebSocketPusher interface {
	Push(userID string, n notification.Notification)
}

type EventPublisher interface {
	Publish(subject, eventType string, data any) error
}

type Service struct {
	storage notification.Storage
	email   DeliveryAdapter
	sms     DeliveryAdapter
	ws      WebSocketPusher
	pub     EventPublisher
	log     *zap.Logger
}

func NewService(storage notification.Storage, email, sms DeliveryAdapter, wsPusher WebSocketPusher, pub EventPublisher, log *zap.Logger) *Service {
	return &Service{
		storage: storage,
		email:   email,
		sms:     sms,
		ws:      wsPusher,
		pub:     pub,
		log:     log,
	}
}

type CreateNotificationRequest struct {
	UserID  string            `json:"user_id"`
	Type    notification.Type `json:"type"`
	Title   string            `json:"title"`
	Message string            `json:"message"`
}

type BulkNotificationRequest struct {
	UserIDs []string          `json:"user_ids"`
	Type    notification.Type `json:"type"`
	Title   string            `json:"title"`
	Message string            `json:"message"`
}

type BulkResult struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

func (s *Service) CreateNotification(ctx context.Context, req CreateNotificationRequest) (notification.Notification, error) {
	n := notification.Notification{
		UserID:  req.UserID,
		Type:    req.Type,
		Title:   req.Title,
		Message: req.Message,
	}

	prefs, err := s.storage.GetPreferences(ctx, req.UserID)
	if err != nil {
		s.log.Warn("Failed to get preferences, delivering anyway", zap.Error(err))
	}

	pref := findPreference(prefs, req.Type)

	if pref != nil && !pref.InApp {
		s.log.Info("In-app notification disabled by preference, skipping",
			zap.String("user_id", req.UserID),
			zap.String("type", string(req.Type)),
		)
		s.deliverExternal(ctx, n, pref)
		return notification.Notification{}, nil
	}

	created, err := s.storage.CreateNotification(ctx, n)
	if err != nil {
		return notification.Notification{}, fmt.Errorf("failed to create notification: %w", err)
	}

	s.log.Info("Notification created",
		zap.String("notification_id", created.ID),
		zap.String("user_id", created.UserID),
		zap.String("type", string(created.Type)),
	)

	if s.ws != nil {
		s.ws.Push(created.UserID, created)
	}

	if s.pub != nil {
		if err := s.pub.Publish("notification.created", "notification.created", map[string]any{
			"notification_id": created.ID,
			"user_id":         created.UserID,
			"type":            created.Type,
			"title":           created.Title,
			"message":         created.Message,
		}); err != nil {
			s.log.Debug("Failed to publish notification.created", zap.Error(err))
		}
	}

	s.deliverExternal(ctx, created, pref)

	return created, nil
}

func (s *Service) GetNotificationByID(ctx context.Context, id string) (notification.Notification, error) {
	return s.storage.GetNotificationByID(ctx, id)
}

func (s *Service) ListNotifications(ctx context.Context, filter notification.Filter, sort pagination.Sort, page pagination.Page) ([]notification.Notification, int, error) {
	return s.storage.ListNotifications(ctx, filter, sort, page)
}

func (s *Service) MarkAsRead(ctx context.Context, id string) (notification.Notification, error) {
	updated, err := s.storage.MarkAsRead(ctx, id)
	if err != nil {
		return notification.Notification{}, err
	}

	s.log.Info("Notification marked as read",
		zap.String("notification_id", updated.ID),
		zap.String("user_id", updated.UserID),
	)

	return updated, nil
}

func (s *Service) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	return s.storage.GetUnreadCount(ctx, userID)
}

func (s *Service) GetUnreadCountsAll(ctx context.Context) ([]notification.UserUnreadCount, error) {
	return s.storage.GetUnreadCountsAll(ctx)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) ([]notification.Preference, error) {
	return s.storage.GetPreferences(ctx, userID)
}

func (s *Service) UpdatePreference(ctx context.Context, pref notification.Preference) (notification.Preference, error) {
	updated, err := s.storage.UpsertPreference(ctx, pref)
	if err != nil {
		return notification.Preference{}, fmt.Errorf("failed to update preference: %w", err)
	}

	s.log.Info("Preference updated",
		zap.String("user_id", updated.UserID),
		zap.String("type", string(updated.Type)),
		zap.Bool("in_app", updated.InApp),
		zap.Bool("email", updated.Email),
		zap.Bool("sms", updated.SMS),
	)

	return updated, nil
}

func (s *Service) BulkCreate(ctx context.Context, req BulkNotificationRequest) (BulkResult, error) {
	result := BulkResult{Total: len(req.UserIDs)}

	for _, userID := range req.UserIDs {
		_, err := s.CreateNotification(ctx, CreateNotificationRequest{
			UserID:  userID,
			Type:    req.Type,
			Title:   req.Title,
			Message: req.Message,
		})
		if err != nil {
			s.log.Warn("Failed to create notification for user in bulk",
				zap.String("user_id", userID),
				zap.Error(err),
			)
			result.Failed++
			continue
		}
		result.Success++
	}

	return result, nil
}

func (s *Service) deliverExternal(ctx context.Context, n notification.Notification, pref *notification.Preference) {
	sendEmail := true
	sendSMS := false
	if pref != nil {
		sendEmail = pref.Email
		sendSMS = pref.SMS
	}

	if sendEmail && s.email != nil {
		if err := s.email.Send(ctx, n); err != nil {
			s.log.Warn("Failed to send email notification", zap.Error(err))
		}
	}

	if sendSMS && s.sms != nil {
		if err := s.sms.Send(ctx, n); err != nil {
			s.log.Warn("Failed to send SMS notification", zap.Error(err))
		}
	}
}

func findPreference(prefs []notification.Preference, nType notification.Type) *notification.Preference {
	for i := range prefs {
		if prefs[i].Type == nType {
			return &prefs[i]
		}
	}
	return nil
}
