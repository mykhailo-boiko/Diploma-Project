package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
)

const (
	StatusSuccess = "success"
	StatusPartial = "partial"
	StatusFailed  = "failed"
)

type Logger struct {
	pool        *pgxpool.Pool
	serviceName string
	log         *zap.Logger
}

func New(pool *pgxpool.Pool, serviceName string, log *zap.Logger) *Logger {
	return &Logger{pool: pool, serviceName: serviceName, log: log.Named("audit")}
}

type Entry struct {
	Action       string
	EntityType   string
	EntityIDs    []string
	Params       any
	ResultStatus string
	SuccessCount int
	FailureCount int
	ErrorMessage string
}

func (l *Logger) Log(ctx context.Context, e Entry) {
	if l == nil || l.pool == nil {
		return
	}

	actorID := middleware.GetUserID(ctx)
	actorRole := middleware.GetUserRole(ctx)
	actorEmail := middleware.GetUserEmail(ctx)
	if actorID == "" {
		actorID = "system"
	}
	if actorEmail == "" {
		actorEmail = "system@chainorchestra.local"
	}
	if actorRole == "" {
		actorRole = "system"
	}

	paramsSnip := ""
	if e.Params != nil {
		b, err := json.Marshal(e.Params)
		if err == nil {
			s := string(b)
			if len(s) > 500 {
				s = s[:500] + "...(truncated)"
			}
			paramsSnip = s
		}
	}

	if e.ResultStatus == "" {
		e.ResultStatus = StatusSuccess
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := l.pool.Exec(bgCtx, `
			INSERT INTO audit.action_log
				(actor_user_id, actor_email, actor_role, service_name, action,
				 entity_type, entity_ids, params_snip, result_status,
				 success_count, failure_count, error_message)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			actorID, actorEmail, actorRole, l.serviceName, e.Action,
			nullIfEmpty(e.EntityType),
			e.EntityIDs, nullIfEmpty(paramsSnip),
			e.ResultStatus, e.SuccessCount, e.FailureCount,
			nullIfEmpty(e.ErrorMessage),
		)
		if err != nil {
			l.log.Warn("Failed to write audit entry",
				zap.String("action", e.Action),
				zap.Error(err),
			)
		}
	}()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
