package gateway

import (
	"net/http"
	"strings"

	"github.com/haradrim/chainorchestra/internal/pkg/auth"
	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
)

type JWTMiddleware struct {
	validator *auth.Validator
}

func NewJWTMiddleware(validator *auth.Validator) *JWTMiddleware {
	return &JWTMiddleware{validator: validator}
}

func (m *JWTMiddleware) Middleware(skipPrefixes []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, prefix := range skipPrefixes {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				httpresponse.Unauthorized(w, "missing_token", "authorization token is required")
				return
			}

			claims, err := m.validator.ValidateAccessToken(tokenStr)
			if err != nil {
				httpresponse.Unauthorized(w, "invalid_token", "invalid or expired token")
				return
			}

			r.Header.Set("X-User-ID", claims.UserID)
			r.Header.Set("X-User-Role", claims.Role)

			next.ServeHTTP(w, r)
		})
	}
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return h[7:]
	}
	return ""
}
