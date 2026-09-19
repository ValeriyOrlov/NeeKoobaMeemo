package handlers

import (
	"log"
	"net/http"
	"os"
	"strings"
)

func InternalAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ✅ TrimSpace убирает \r, \n и пробелы
		expectedSecret := strings.TrimSpace(os.Getenv("INTERNAL_SECRET"))
		incomingSecret := strings.TrimSpace(r.Header.Get("X-Internal-Secret"))

		// %q выведет строку в кавычках с экранированием (например: "b1726... \r")
		log.Printf("🔍 [InternalAuth] Expected: %q | Incoming: %q", expectedSecret, incomingSecret)

		if expectedSecret == "" || incomingSecret != expectedSecret {
			http.Error(w, `{"error": "unauthorized internal request"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}
