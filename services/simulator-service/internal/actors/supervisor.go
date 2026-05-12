package actors

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/simulator-service/internal/state"
)

func RunSupervisor(ctx context.Context, st *state.State, log *zap.Logger) {
	log = log.Named("supervisor")
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	var lastErrors, lastTicks int64
	var lastErrSnapshot time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !st.Enabled() {
				lastErrors = st.Counters.Errors.Load()
				continue
			}
			currErrors := st.Counters.Errors.Load()
			currTicks := st.Counters.OrdersCreated.Load() +
				st.Counters.OrdersProgressed.Load() +
				st.Counters.ShipmentsAdvanced.Load() +
				st.Counters.StockAdjustments.Load() +
				st.Counters.NotificationsSent.Load()

			deltaErrors := currErrors - lastErrors
			deltaTicks := currTicks - lastTicks

			lastErrors = currErrors
			lastTicks = currTicks

			if deltaTicks < 5 {
				continue
			}
			rate := float64(deltaErrors) / float64(deltaTicks+deltaErrors)
			if rate > 0.30 {
				if time.Since(lastErrSnapshot) > time.Minute {
					log.Warn("High error rate, auto-stopping simulator",
						zap.Float64("rate", rate),
						zap.Int64("errors_15s", deltaErrors),
						zap.Int64("actions_15s", deltaTicks),
					)
					st.Stop()
					lastErrSnapshot = time.Now()
				}
			}
		}
	}
}
