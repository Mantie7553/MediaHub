package auth

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

const UserContextKey contextKey = "user"

const (
	maxTokens  = 10
	refillRate = 1.0 / 10.0
)

var rateLimitMap sync.Map

type UserContext struct {
	UserID string
	Role   string
}

type RateLimitEntry struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

/*
Function:	Middleware
Purpose:	Handle authentication checks
Params:
  - next: the http handler that will deal with the actual request
*/
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check that the Authorization header is present
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		// if there is no bearer and token it is invalid
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		// if the token is invalid then the request is invalid
		claims, err := ValidateToken(parts[1])
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{
			UserID: claims.UserID,
			Role:   claims.Role,
		})

		// move onto the actual request handling
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

/*
Function:	RequireAdmin
Purpose:	Only allow access for admin users
Params:
  - next: the http handler that will deal with the actual request
*/
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(UserContextKey).(*UserContext)
		if !ok || user.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

/*
Function:	GetUser
Purpose:	Get the current user
Params:
  - r: http request coming from the frontend
*/
func GetUser(r *http.Request) *UserContext {
	user, _ := r.Context().Value(UserContextKey).(*UserContext)
	return user
}

func getClientIP(r *http.Request) string {
	var addr string
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		addr = strings.TrimSpace(strings.Split(xff, ",")[0])
	} else {
		addr = r.RemoteAddr
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}

	return host
}

func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		actual, _ := rateLimitMap.LoadOrStore(ip, &RateLimitEntry{tokens: maxTokens, lastSeen: time.Now()})
		entry := actual.(*RateLimitEntry)
		entry.mu.Lock()
		defer entry.mu.Unlock()
		elapsed := time.Since(entry.lastSeen).Seconds()
		entry.tokens = min(entry.tokens+elapsed*refillRate, maxTokens)
		entry.lastSeen = time.Now()
		entry.tokens--
		if entry.tokens < 0 {
			entry.tokens = 0
			http.Error(w, "too many requests", http.StatusTooManyRequests)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
