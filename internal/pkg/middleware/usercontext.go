package middleware

import (
	"context"
	"net/http"
)

const (
	userIDHeader    = "X-User-ID"
	userRoleHeader  = "X-User-Role"
	userEmailHeader = "X-User-Email"

	userIDKey    contextKey = "user_id"
	userRoleKey  contextKey = "user_role"
	userEmailKey contextKey = "user_email"
)

func UserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if id := r.Header.Get(userIDHeader); id != "" {
			ctx = context.WithValue(ctx, userIDKey, id)
		}
		if role := r.Header.Get(userRoleHeader); role != "" {
			ctx = context.WithValue(ctx, userRoleKey, role)
		}
		if email := r.Header.Get(userEmailHeader); email != "" {
			ctx = context.WithValue(ctx, userEmailKey, email)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

func GetUserRole(ctx context.Context) string {
	role, _ := ctx.Value(userRoleKey).(string)
	return role
}

func GetUserEmail(ctx context.Context) string {
	email, _ := ctx.Value(userEmailKey).(string)
	return email
}

func WithUserContext(ctx context.Context, userID, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, userRoleKey, role)
	return ctx
}
