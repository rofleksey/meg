package httpx

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestRequestIDNormalizesHeadersWithoutMutatingCaller(t *testing.T) {
	const (
		validID    = "request_0123456789abcdef"
		headerName = "X-Request-Id"
	)

	tests := []struct {
		name   string
		header http.Header
		wantID string
	}{
		{name: "nil headers"},
		{name: "missing header", header: make(http.Header)},
		{name: "unsafe header", header: http.Header{headerName: {"bad\nvalue"}}},
		{name: "valid header", header: http.Header{headerName: {validID}}, wantID: validID},
		{name: "multiple values", header: http.Header{headerName: {validID, "bad\nvalue"}}, wantID: validID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			request.Header = test.header
			originalHeader := request.Header.Clone()
			response := httptest.NewRecorder()
			handler := RequestID(http.HandlerFunc(func(writer http.ResponseWriter, downstream *http.Request) {
				id := RequestIDFrom(downstream.Context())
				if !ValidRequestID(id) || (test.wantID != "" && id != test.wantID) {
					t.Fatalf("context ID = %q, expected valid ID %q", id, test.wantID)
				}

				values := downstream.Header.Values("X-Request-ID")
				if len(values) != 1 || values[0] != id || writer.Header().Get("X-Request-ID") != id {
					t.Fatalf("request headers = %#v, response headers = %#v, context ID = %q", downstream.Header, writer.Header(), id)
				}

				downstream.Header.Set("X-Downstream", "changed")
			}))
			handler.ServeHTTP(response, request)

			if !reflect.DeepEqual(request.Header, originalHeader) || RequestIDFrom(request.Context()) != "" {
				t.Fatalf("caller request was changed: headers = %#v, context ID = %q", request.Header, RequestIDFrom(request.Context()))
			}
		})
	}
}

func TestValidRequestID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{id: ""},
		{id: strings.Repeat("a", 15)},
		{id: strings.Repeat("a", 16), valid: true},
		{id: strings.Repeat("a", 128), valid: true},
		{id: strings.Repeat("a", 129)},
		{id: "ABCdef0123456789_-", valid: true},
		{id: "bad\n0123456789abcdef"},
		{id: "bad 0123456789abcdef"},
		{id: "éééééééééééééééé"},
	}
	for _, test := range tests {
		if got := ValidRequestID(test.id); got != test.valid {
			t.Errorf("ValidRequestID(%q) = %t, want %t", test.id, got, test.valid)
		}
	}
}
