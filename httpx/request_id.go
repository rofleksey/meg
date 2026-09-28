// Package httpx provides small, transport-only HTTP helpers.
package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

type requestIDKey struct{}

// RequestID validates an inbound request ID or issues a new one, then makes
// it available in the downstream request header, response header, and request
// context. The caller's request headers are left unchanged.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !ValidRequestID(id) {
			id = newRequestID()
		}

		w.Header().Set("X-Request-ID", id)
		request := r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id))

		request.Header = r.Header.Clone()
		if request.Header == nil {
			request.Header = make(http.Header)
		}

		request.Header.Set("X-Request-ID", id)
		next.ServeHTTP(w, request)
	})
}

func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// ValidRequestID accepts bounded ASCII identifiers suitable for correlation.
func ValidRequestID(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}

	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '-' && character != '_' {
			return false
		}
	}

	return true
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err == nil {
		return hex.EncodeToString(bytes)
	}

	return strings.ReplaceAll(time.Now().UTC().Format("20060102T150405.000000000"), ".", "")
}
