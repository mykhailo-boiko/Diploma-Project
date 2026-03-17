package delivery

import (
	"context"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
)

type EmailAdapter struct {
	log *zap.Logger
}

func NewEmailAdapter(log *zap.Logger) *EmailAdapter {
	return &EmailAdapter{log: log}
}

func (a *EmailAdapter) Send(_ context.Context, n notification.Notification) error {
	a.log.Info("Mock email sent",
		zap.String("to_user", n.UserID),
		zap.String("subject", n.Title),
		zap.String("body", n.Message),
		zap.String("type", string(n.Type)),
	)
	return nil
}

type SMSAdapter struct {
	log *zap.Logger
}

func NewSMSAdapter(log *zap.Logger) *SMSAdapter {
	return &SMSAdapter{log: log}
}

func (a *SMSAdapter) Send(_ context.Context, n notification.Notification) error {
	a.log.Info("Mock SMS sent",
		zap.String("to_user", n.UserID),
		zap.String("message", n.Message),
		zap.String("type", string(n.Type)),
	)
	return nil
}
