package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDRejectsUnsafeHeaderAndPropagatesGeneratedID(t *testing.T) {
	var observed string
	handler := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		observed = RequestIDFrom(request.Context())
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "bad\nvalue")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	id := response.Header().Get("X-Request-ID")
	if !ValidRequestID(id) || id == "bad\nvalue" || observed != id {
		t.Fatalf("response ID = %q, context ID = %q", id, observed)
	}
}
