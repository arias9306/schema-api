// Package mock renders and serves mock endpoint responses from
// schema-defined templates.
package mock

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/arias9306/schema-api/fakegen"
	"github.com/arias9306/schema-api/schema"
)

type Handler struct {
	endpoints []schema.Endpoint
	bodyRefs  []bool
}

func NewHandler(endpoints []schema.Endpoint) *Handler {
	bodyRefs := make([]bool, len(endpoints))
	for i, e := range endpoints {
		bodyRefs[i] = referencesBody(e.Response)
	}
	return &Handler{endpoints: endpoints, bodyRefs: bodyRefs}
}

func (h *Handler) Register(mux *http.ServeMux) (err error) {
	seen := make(map[string]bool, len(h.endpoints))
	for _, e := range h.endpoints {
		pattern := strings.ToUpper(strings.TrimSpace(e.Method)) + " " + e.Path
		if seen[pattern] {
			return fmt.Errorf("duplicate endpoint pattern %q", pattern)
		}
		seen[pattern] = true
	}

	for i := range h.endpoints {
		pattern := strings.ToUpper(strings.TrimSpace(h.endpoints[i].Method)) + " " + h.endpoints[i].Path

		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("registering endpoint %q: %v", pattern, r)
				}
			}()

			mux.HandleFunc(pattern, h.handlerFor(i))
		}()

		if err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) handlerFor(i int) http.HandlerFunc {
	e := h.endpoints[i]

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := map[string]string{}

		for _, name := range e.ParamNames() {
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

		if h.bodyRefs[i] {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				for name, value := range body {
					ctx["body."+name] = fmt.Sprintf("%v", value)
				}
			}
		}

		status := e.Status
		if status == 0 {
			status = http.StatusOK
		}

		for name, value := range e.Headers {
			w.Header().Set(name, value)
		}

		rendered, err := Render(e.Response, rand.New(rand.NewSource(time.Now().UnixNano())), ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "rendering response: %v", err)
			return
		}

		writeJSON(w, status, rendered)
	}
}

func referencesBody(template any) bool {
	data, err := json.Marshal(template)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "{{body")
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf(format, args...)})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

type renderer struct {
	rand *rand.Rand
	ctx  map[string]string
	now  time.Time
}

func Render(template any, randomizer *rand.Rand, ctx map[string]string) (any, error) {
	render := &renderer{rand: randomizer, ctx: ctx, now: time.Now()}
	return render.render(template, "")
}

func (r *renderer) render(value any, name string) (any, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil

	case string:
		return r.interpolate(v), nil

	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			rendered, err := r.render(item, "")
			if err != nil {
				return nil, err
			}
			out[i] = rendered
		}
		return out, nil

	case map[string]any:
		if t, ok := v["type"].(string); ok {
			return r.renderSpec(t, v, name)
		}

		out := make(map[string]any, len(v))
		for key, val := range v {
			rendered, err := r.render(val, key)
			if err != nil {
				return nil, err
			}
			out[key] = rendered
		}
		return out, nil

	default:
		return v, nil
	}
}

func (r *renderer) renderSpec(t string, v map[string]any, name string) (any, error) {
	switch t {
	case "array":
		count := 1
		if c, ok := plainInt(v["count"]); ok && c > 0 {
			count = c
		}

		items := v["items"]
		out := make([]any, 0, count)
		for i := 0; i < count; i++ {
			item, err := r.render(items, "")
			if err != nil {
				return nil, err
			}
			out = append(out, item)
		}
		return out, nil

	case "object":
		properties, _ := v["properties"].(map[string]any)
		out := make(map[string]any, len(properties))
		for key, val := range properties {
			rendered, err := r.render(val, key)
			if err != nil {
				return nil, err
			}
			out[key] = rendered
		}
		return out, nil

	default:
		spec := specFromMap(v)
		if spec.Type == "string" && spec.Format == "" && name != "" {
			if format := fakegen.ResolveFormat(name); format != "" {
				spec.Format = format
			}
		}

		val, err := fakegen.Value(r.rand, spec)
		if err != nil {
			return nil, err
		}
		return val, nil
	}
}

func specFromMap(value map[string]any) fakegen.Spec {
	spec := fakegen.Spec{}

	if t, ok := value["type"].(string); ok {
		spec.Type = t
	}

	if f, ok := floatValue(value["min"]); ok {
		spec.Min = f
	}

	if f, ok := floatValue(value["max"]); ok {
		spec.Max = f
	}

	if n, ok := intValue(value["min_length"]); ok {
		spec.MinLength = n
	}

	if n, ok := intValue(value["max_length"]); ok {
		spec.MaxLength = n
	}

	if rg, ok := value["regex"].(string); ok {
		spec.Regex = rg
	}

	if format, ok := value["format"].(string); ok {
		spec.Format = format
	}

	if def, ok := value["default"]; ok {
		spec.Default = def
	}

	return spec
}

func floatValue(value any) (*float64, bool) {
	switch n := value.(type) {
	case float64:
		return &n, true
	case int:
		f := float64(n)
		return &f, true
	}
	return nil, false
}

func intValue(value any) (*int, bool) {
	n, ok := plainInt(value)
	if !ok {
		return nil, false
	}
	return &n, true
}

func plainInt(value any) (int, bool) {
	switch n := value.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

func (r *renderer) interpolate(s string) string {
	if !strings.Contains(s, "{{") {
		return s
	}

	var buf strings.Builder
	rest := s

	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			buf.WriteString(rest)
			break
		}

		buf.WriteString(rest[:start])

		end := strings.Index(rest[start:], "}}")
		if end < 0 {
			buf.WriteString(rest[start:])
			break
		}

		expr := strings.TrimSpace(rest[start+2 : start+end])
		buf.WriteString(r.lookup(expr))

		rest = rest[start+end+2:]
	}

	return buf.String()
}

func (r *renderer) lookup(expr string) string {
	if expr == "now" {
		return r.now.Format(time.RFC3339)
	}

	return r.ctx[expr]
}
