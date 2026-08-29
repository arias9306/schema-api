// Package httputil provides shared HTTP response helpers used by the
// schema-api handlers.
package httputil

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func WriteError(w http.ResponseWriter, status int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	WriteJSON(w, status, map[string]string{"error": msg})
}
