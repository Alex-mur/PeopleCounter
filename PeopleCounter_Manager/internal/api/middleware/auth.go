package middleware

import (
	"net/http"

	"github.com/go-chi/jwtauth/v5"
)

func RequireAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, claims, _ := jwtauth.FromContext(r.Context())
		if claims["type"] != "access" {
			http.Error(w, "Требуется Access Token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
