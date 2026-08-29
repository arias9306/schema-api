// Package httputil provides shared HTTP response helpers used by the
// schema-api handlers.
package httputil

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

func RouteKey(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + path
}

func BuildRequestContext(r *http.Request, paramNames []string, includeBody bool) map[string]string {
	ctx := map[string]string{}

	for _, name := range paramNames {
		ctx["path."+name] = r.PathValue(name)
	}

	for name, values := range r.URL.Query() {
		if len(values) > 0 {
			ctx["query."+name] = values[0]
		}
	}

	for name, values := range r.Header {
		if len(values) > 0 {
			ctx["header."+strings.ToLower(name)] = values[0]
		}
	}

	if includeBody {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			for name, value := range body {
				ctx["body."+name] = fmt.Sprintf("%v", value)
			}
		}
	}

	return ctx
}
