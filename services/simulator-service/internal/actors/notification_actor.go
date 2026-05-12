package actors

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/simulator-service/internal/httpclient"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/metrics"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/state"
)

type NotificationActor struct {
	hc    *httpclient.Client
	state *state.State
	log   *zap.Logger
	rng   *rand.Rand
	mu    sync.Mutex

	usersCache    []userLite
	usersFetched  time.Time
}

type userLite struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func NewNotificationActor(hc *httpclient.Client, st *state.State, log *zap.Logger) *NotificationActor {
	return &NotificationActor{
		hc:    hc,
		state: st,
		log:   log,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano() + 4)),
	}
}

func (a *NotificationActor) Name() string { return "notification_actor" }

func (a *NotificationActor) Tick(ctx context.Context) error {
	if err := a.ensureUsers(ctx); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.usersCache) == 0 {
		return nil
	}
	target := a.usersCache[a.rng.Intn(len(a.usersCache))]

	templates := []struct {
		typ      string
		title    string
		message  string
		priority string
	}{
		{"system_alert", "System health check", "All services responded within SLA.", "low"},
		{"info", "Daily ops summary", "Pipeline throughput is on baseline; no escalations.", "low"},
		{"low_stock_alert", "Watchlist update", fmt.Sprintf("%d SKUs need replenishment review", 5+a.rng.Intn(20)), "medium"},
		{"analytics_anomaly", "AOV variation", "Average order value drifted slightly versus 30d baseline.", "medium"},
		{"shipment_update", "Carrier digest", "On-time rate aligned with this week's benchmark.", "low"},
	}
	t := templates[a.rng.Intn(len(templates))]

	body := map[string]any{
		"user_id":  target.ID,
		"type":     t.typ,
		"title":    t.title,
		"message":  t.message,
		"priority": t.priority,
		"channel":  "in_app",
	}
	if err := a.hc.Post(ctx, "/api/v1/notifications", body, nil); err != nil {
		return err
	}

	a.state.Counters.NotificationsSent.Add(1)
	metrics.NotificationsCreated.Inc()
	return nil
}

func (a *NotificationActor) ensureUsers(ctx context.Context) error {
	a.mu.Lock()
	fresh := time.Since(a.usersFetched) < 10*time.Minute && len(a.usersCache) > 0
	a.mu.Unlock()
	if fresh {
		return nil
	}
	q := url.Values{}
	q.Set("limit", "100")
	var env listEnvelope[userLite]
	if err := a.hc.Get(ctx, "/api/v1/users", q, &env); err != nil {
		return err
	}
	a.mu.Lock()
	a.usersCache = env.Data
	a.usersFetched = time.Now()
	a.mu.Unlock()
	return nil
}
