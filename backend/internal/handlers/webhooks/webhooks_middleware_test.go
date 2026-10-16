package webhooks

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireSecret(t *testing.T) {
	handler := RequireSecret(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		env    string
		header string
		query  string
		status int
	}{
		{"valid", "s3cret", "s3cret", "", http.StatusOK},
		{"missing header", "s3cret", "", "", http.StatusUnauthorized},
		{"wrong header", "s3cret", "nope", "", http.StatusUnauthorized},
		{"query param ignored", "s3cret", "", "?secret=s3cret", http.StatusUnauthorized},
		{"unset fails closed", "", "", "", http.StatusServiceUnavailable},
		{"unset rejects any header", "", "anything", "", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WEBHOOK_SECRET", tt.env)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/sonarr"+tt.query, nil)
			if tt.header != "" {
				req.Header.Set(SecretHeader, tt.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
		})
	}
}
