package actors

import (
	"context"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/simulator-service/internal/metrics"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/state"
)

type Actor interface {
	Name() string
	Tick(ctx context.Context) error
}

func RunActor(ctx context.Context, a Actor, st *state.State, intervalFn func(ScenarioTuning) float64, log *zap.Logger) {
	name := a.Name()
	log = log.With(zap.String("actor", name))
	log.Info("Actor started")

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(time.Now().Nanosecond())))

	for {
		select {
		case <-ctx.Done():
			log.Info("Actor stopped")
			return
		default:
		}

		baseSec := 0.0
		if st.Enabled() {
			tuning := TuningFor(st.Scenario())
			baseSec = intervalFn(tuning)
		}

		if !st.Enabled() || baseSec <= 0 {
			sleepOrCancel(ctx, 2*time.Second)
			continue
		}

		speed := st.Speed()
		if speed <= 0 {
			speed = 1
		}
		effective := baseSec / speed
		jitter := 0.8 + rng.Float64()*0.4
		wait := time.Duration(effective * jitter * float64(time.Second))
		if wait < 250*time.Millisecond {
			wait = 250 * time.Millisecond
		}

		sleepOrCancel(ctx, wait)

		if !st.Enabled() {
			continue
		}

		start := time.Now()
		tickCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := a.Tick(tickCtx)
		cancel()
		duration := time.Since(start).Seconds()

		metrics.ActorTicks.WithLabelValues(name).Inc()
		metrics.ActorTickDuration.WithLabelValues(name).Observe(duration)

		if err != nil {
			metrics.ActorErrors.WithLabelValues(name).Inc()
			st.Counters.Errors.Add(1)
			log.Warn("Tick failed", zap.Error(err))
		}
	}
}

func sleepOrCancel(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
