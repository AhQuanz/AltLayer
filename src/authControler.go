package src

import "net/http"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip middleware for the /login route
		if r.URL.Path == "/login" {
			next.ServeHTTP(w, r)
			return
		}

		// Example: Check for an Authorization header
		token := r.Header.Get("Token")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		_, isValid := CheckTokenExpiry(token)
		if !isValid {
			http.Error(w, "Token Not Valid", http.StatusUnauthorized)
			return
		}
		// Call the next handler if authenticated
		next.ServeHTTP(w, r)
	})
}
