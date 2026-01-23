package middleware

import (
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
)

func Recovery(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("Panic recovered",
						zap.Any("error", err),
						zap.String("stack", string(debug.Stack())),
						zap.String("request_id", GetRequestID(r.Context())),
					)
					httpresponse.InternalError(w, "internal_error", "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
