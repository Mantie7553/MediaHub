package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

type UserContext struct {
	UserID string
	Role   string
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
