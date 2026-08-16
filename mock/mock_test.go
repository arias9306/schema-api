package mock

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arias9306/schema-api/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderLiterals(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	ctx := map[string]string{}

	v, err := Render("hello", rng, ctx)
	require.NoError(t, err)
	assert.Equal(t, "hello", v)

	v, err = Render(float64(3.5), rng, ctx)
	require.NoError(t, err)
	assert.Equal(t, 3.5, v)

	v, err = Render(true, rng, ctx)
	require.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = Render(nil, rng, ctx)
	require.NoError(t, err)
	assert.Nil(t, v)

	v, err = Render([]any{"a", float64(1), false}, rng, ctx)
	require.NoError(t, err)
	assert.Equal(t, []any{"a", float64(1), false}, v)
}

func TestRenderInterpolation(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	ctx := map[string]string{
		"path.id":       "42",
		"query.page":    "3",
		"header.x-mock": "yes",
		"body.name":     "bob",
	}

	v, err := Render("{{path.id}} {{query.page}} {{header.x-mock}} {{body.name}}", rng, ctx)
	require.NoError(t, err)
	assert.Equal(t, "42 3 yes bob", v)
}

func TestRenderMissingKeysBecomeEmpty(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	v, err := Render("{{path.id}}-{{nope}}", rng, map[string]string{})
	require.NoError(t, err)
	assert.Equal(t, "-", v)
}

func TestRenderNow(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	v, err := Render("{{now}}", rng, map[string]string{})
	require.NoError(t, err)
	_, err = time.Parse(time.RFC3339, v.(string))
	require.NoError(t, err)
}

func TestRenderObjectTemplate(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	template := map[string]any{
		"user":   "{{path.id}}",
		"fixed":  "literal",
		"nested": map[string]any{"deep": "{{query.q}}"},
	}
	ctx := map[string]string{"path.id": "7", "query.q": "x"}

	v, err := Render(template, rng, ctx)
	require.NoError(t, err)

	got := v.(map[string]any)
	assert.Equal(t, "7", got["user"])
	assert.Equal(t, "literal", got["fixed"])
	assert.Equal(t, map[string]any{"deep": "x"}, got["nested"])
}

func TestRenderArraySpec(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	ctx := map[string]string{}

	item := map[string]any{"type": "int", "min": 1.0, "max": 1.0}

	t.Run("count", func(t *testing.T) {
		v, err := Render(map[string]any{"type": "array", "count": 3, "items": item}, rng, ctx)
		require.NoError(t, err)
		assert.Equal(t, []any{1, 1, 1}, v)
	})

	t.Run("float count", func(t *testing.T) {
		v, err := Render(map[string]any{"type": "array", "count": 2.0, "items": item}, rng, ctx)
		require.NoError(t, err)
		assert.Equal(t, []any{1, 1}, v)
	})

	t.Run("zero or negative count defaults to one", func(t *testing.T) {
		v, err := Render(map[string]any{"type": "array", "count": 0, "items": item}, rng, ctx)
		require.NoError(t, err)
		assert.Equal(t, []any{1}, v)
	})

	t.Run("item error propagates", func(t *testing.T) {
		_, err := Render(map[string]any{"type": "array", "count": 2, "items": map[string]any{"type": "blob"}}, rng, ctx)
		require.Error(t, err)
		assert.ErrorContains(t, err, `unsupported type "blob"`)
	})
}

func TestRenderObjectSpec(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	template := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "min_length": 5.0, "max_length": 5.0},
			"age":  map[string]any{"type": "int", "min": 1.0, "max": 1.0},
		},
	}

	v, err := Render(template, rng, map[string]string{})
	require.NoError(t, err)

	got := v.(map[string]any)
	assert.Len(t, got["name"].(string), 5)
	assert.Equal(t, 1, got["age"])
}

func TestRenderFormatHeuristicByPropertyName(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	template := map[string]any{"email": map[string]any{"type": "string"}}

	v, err := Render(template, rng, map[string]string{})
	require.NoError(t, err)
	assert.Contains(t, v.(map[string]any)["email"].(string), "@")
}

func TestRenderScalarTypes(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	ctx := map[string]string{}

	v, err := Render(map[string]any{"type": "bool"}, rng, ctx)
	require.NoError(t, err)
	_, ok := v.(bool)
	require.True(t, ok)

	v, err = Render(map[string]any{"type": "float"}, rng, ctx)
	require.NoError(t, err)
	_, ok = v.(float64)
	require.True(t, ok)

	v, err = Render(map[string]any{"type": "datetime"}, rng, ctx)
	require.NoError(t, err)
	s, ok := v.(string)
	require.True(t, ok)
	_, err = time.Parse(time.RFC3339, s)
	require.NoError(t, err)

	v, err = Render(map[string]any{"type": "int"}, rng, ctx)
	require.NoError(t, err)
	n, ok := v.(int)
	require.True(t, ok)
	assert.GreaterOrEqual(t, n, 0)
	assert.LessOrEqual(t, n, 10000)
}

func TestRenderUnsupportedType(t *testing.T) {
	_, err := Render(map[string]any{"type": "blob"}, rand.New(rand.NewSource(1)), map[string]string{})
	require.Error(t, err)
	assert.ErrorContains(t, err, `unsupported type "blob"`)
}

func TestRenderDeterministicWithSeed(t *testing.T) {
	template := map[string]any{"email": map[string]any{"type": "string", "format": "email"}}

	a, err := Render(template, rand.New(rand.NewSource(42)), map[string]string{})
	require.NoError(t, err)
	b, err := Render(template, rand.New(rand.NewSource(42)), map[string]string{})
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestSpecFromMap(t *testing.T) {
	min, max := 1.0, 10.0
	minLen, maxLen := 3, 9

	spec := specFromMap(map[string]any{
		"type":       "string",
		"min":        1,
		"max":        10.0,
		"min_length": 3,
		"max_length": 9.0,
		"regex":      "^x$",
		"format":     "email",
		"default":    "d",
	})

	assert.Equal(t, "string", spec.Type)
	assert.Equal(t, &min, spec.Min)
	assert.Equal(t, &max, spec.Max)
	assert.Equal(t, &minLen, spec.MinLength)
	assert.Equal(t, &maxLen, spec.MaxLength)
	assert.Equal(t, "^x$", spec.Regex)
	assert.Equal(t, "email", spec.Format)
	assert.Equal(t, "d", spec.Default)
}

func TestInterpolate(t *testing.T) {
	r := &renderer{ctx: map[string]string{"a": "1"}}

	assert.Equal(t, "x1y", r.interpolate("x{{a}}y"))
	assert.Equal(t, "1 1", r.interpolate("{{a}} {{a}}"))
	assert.Equal(t, "{{a", r.interpolate("{{a"))
	assert.Equal(t, "xy", r.interpolate("x{{b}}y"))
	assert.Equal(t, "no braces", r.interpolate("no braces"))
}

func TestLookup(t *testing.T) {
	r := &renderer{
		now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		ctx: map[string]string{"path.id": "v"},
	}

	assert.Equal(t, "2026-01-02T03:04:05Z", r.lookup("now"))
	assert.Equal(t, "v", r.lookup("path.id"))
	assert.Equal(t, "", r.lookup("missing"))
}

func TestReferencesBody(t *testing.T) {
	assert.True(t, referencesBody(map[string]any{"echo": "{{body.name}}"}))
	assert.False(t, referencesBody(map[string]any{"echo": "{{query.x}}"}))
	assert.False(t, referencesBody("plain"))
}

func TestRegister(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		h := NewHandler([]schema.Endpoint{
			{Method: "GET", Path: "/health", Response: map[string]any{"ok": true}},
		})
		require.NoError(t, h.Register(http.NewServeMux()))
	})

	t.Run("duplicate pattern", func(t *testing.T) {
		h := NewHandler([]schema.Endpoint{
			{Method: "GET", Path: "/x", Response: "a"},
			{Method: "get", Path: "/x", Response: "b"},
		})
		err := h.Register(http.NewServeMux())
		require.Error(t, err)
		assert.ErrorContains(t, err, `duplicate endpoint pattern "GET /x"`)
	})

	t.Run("conflicting patterns", func(t *testing.T) {
		h := NewHandler([]schema.Endpoint{
			{Method: "GET", Path: "/{id}", Response: "a"},
			{Method: "GET", Path: "/{name}", Response: "b"},
		})
		err := h.Register(http.NewServeMux())
		require.Error(t, err)
		assert.ErrorContains(t, err, "registering endpoint")
	})
}

func TestHandlerServesRequests(t *testing.T) {
	response := map[string]any{
		"user_id": "{{path.id}}",
		"page":    "{{query.page}}",
		"token":   "{{header.x-token}}",
		"echo":    "{{body.name}}",
		"fixed":   "literal",
		"age":     map[string]any{"type": "int", "min": 1.0, "max": 1.0},
	}

	h := NewHandler([]schema.Endpoint{
		{Method: "GET", Path: "/users/{id}", Status: 201, Headers: map[string]string{"X-Mock": "true"}, Response: response},
		{Method: "GET", Path: "/bad", Response: map[string]any{"type": "blob"}},
		{Method: "GET", Path: "/plain", Response: "hello"},
		{Method: "GET", Path: "/empty", Status: 0, Response: map[string]any{"ok": true}},
	})

	mux := http.NewServeMux()
	require.NoError(t, h.Register(mux))

	t.Run("interpolation and status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/42?page=3", strings.NewReader(`{"name":"bob"}`))
		req.Header.Set("X-Token", "sekret")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, 201, rec.Code)
		assert.Equal(t, "true", rec.Header().Get("X-Mock"))

		var got map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, "42", got["user_id"])
		assert.Equal(t, "3", got["page"])
		assert.Equal(t, "sekret", got["token"])
		assert.Equal(t, "bob", got["echo"])
		assert.Equal(t, "literal", got["fixed"])
		assert.Equal(t, 1, int(got["age"].(float64)))
	})

	t.Run("500 on render failure", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/bad", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		var got map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Contains(t, got["error"], "rendering response")
	})

	t.Run("literal response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/plain", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `"hello"`, rec.Body.String())
	})

	t.Run("default status 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/empty", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})
}
