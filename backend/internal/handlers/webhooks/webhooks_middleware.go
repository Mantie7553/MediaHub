package webhooks

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

const SecretHeader = "X-MediaHub-Secret"

/*
Function:	RequireSecret
Purpose:	Only allow webhook calls carrying the shared secret in the X-MediaHub-Secret header
Params:
  - next: the http handler that will deal with the actual request
*/
func RequireSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv("WEBHOOK_SECRET")
		// fail closed if the server isn't configured
		if secret == "" {
			logger.Error("WEBHOOK_SECRET not set, rejecting webhook")
			http.Error(w, "webhooks not configured", http.StatusServiceUnavailable)
			return
		}

		got := r.Header.Get(SecretHeader)
		if subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
			http.Error(w, "invalid webhook secret", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
