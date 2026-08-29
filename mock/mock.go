// Package mock renders and serves mock endpoint responses from
// schema-defined templates.
package mock

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/arias9306/schema-api/fakegen"
	"github.com/arias9306/schema-api/httputil"
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
		pattern := httputil.RouteKey(e.Method, e.Path)
		if seen[pattern] {
			return fmt.Errorf("duplicate endpoint pattern %q", pattern)
		}
		seen[pattern] = true
	}

	for i := range h.endpoints {
		pattern := httputil.RouteKey(h.endpoints[i].Method, h.endpoints[i].Path)

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
		ctx := httputil.BuildRequestContext(r, e.ParamNames(), h.bodyRefs[i])

		status := cmp.Or(e.Status, http.StatusOK)

		for name, value := range e.Headers {
			w.Header().Set(name, value)
		}

		rendered, err := Render(e.Response, rand.New(rand.NewSource(time.Now().UnixNano())), ctx)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "rendering response: %v", err)
			return
		}

		httputil.WriteJSON(w, status, rendered)
	}
}

func referencesBody(template any) bool {
	data, err := json.Marshal(template)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "{{body")
}

type renderer struct {
	rand *rand.Rand
	ctx  map[string]string
	now  time.Time
}

func Render(template any, rng *rand.Rand, ctx map[string]string) (any, error) {
	render := &renderer{rand: rng, ctx: ctx, now: time.Now()}
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
		if c, ok := fakegen.AsInt(v["count"]); ok && c > 0 {
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
		spec := fakegen.SpecFromMap(v, name)

		val, err := fakegen.Value(r.rand, spec)
		if err != nil {
			return nil, err
		}
		return val, nil
	}
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
