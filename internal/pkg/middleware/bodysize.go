package middleware

import (
	"fmt"
	"net/http"
)

const DefaultMaxBodyBytes = int64(1 << 20)

func BodySize(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				body := fmt.Sprintf(`{"error":{"code":"payload_too_large","message":"request body exceeds the %d byte limit","data":{"max_bytes":%d,"received_content_length":%d,"suggestion":"Break the request into smaller chunks or use bulk endpoints with sensible page sizes."}}}`,
					maxBytes, maxBytes, r.ContentLength)
				_, _ = w.Write([]byte(body))
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
