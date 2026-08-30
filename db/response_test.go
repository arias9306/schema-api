package db

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTableEndpointResponseScalar(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	rows := []map[string]any{
		{"users__name": "Alice", "users__email": "alice@example.com", "users__id": int64(1)},
	}
	template := map[string]any{
		"user_name":  "{{users.name}}",
		"user_email": "{{users.email}}",
	}

	result := BuildTableEndpointResponse(rng, template, rows)

	expected := map[string]any{
		"user_name":  "Alice",
		"user_email": "alice@example.com",
	}
	assert.Equal(t, expected, result)
}

func TestBuildTableEndpointResponseArray(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	rows := []map[string]any{
		{"posts__title": "First"},
		{"posts__title": "Second"},
	}
	template := map[string]any{
		"posts": map[string]any{
			"type":  "array",
			"items": map[string]any{"title": "{{posts.title}}"},
		},
	}

	result := BuildTableEndpointResponse(rng, template, rows)

	expected := map[string]any{
		"posts": []any{
			map[string]any{"title": "First"},
			map[string]any{"title": "Second"},
		},
	}
	assert.Equal(t, expected, result)
}

func TestBuildTableEndpointResponseMixed(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	rows := []map[string]any{
		{"users__name": "Alice", "posts__title": "First"},
		{"users__name": "Alice", "posts__title": "Second"},
	}
	template := map[string]any{
		"user_name": "{{users.name}}",
		"posts": map[string]any{
			"type":  "array",
			"items": map[string]any{"title": "{{posts.title}}"},
		},
	}

	result := BuildTableEndpointResponse(rng, template, rows)

	expected := map[string]any{
		"user_name": "Alice",
		"posts": []any{
			map[string]any{"title": "First"},
			map[string]any{"title": "Second"},
		},
	}
	assert.Equal(t, expected, result)
}

func TestBuildTableEndpointResponseEmptyRows(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	rows := []map[string]any{}
	template := map[string]any{
		"user_name": "{{users.name}}",
		"posts": map[string]any{
			"type":  "array",
			"items": map[string]any{"title": "{{posts.title}}"},
		},
		"literal": "static",
	}

	result := BuildTableEndpointResponse(rng, template, rows)

	expected := map[string]any{
		"user_name": nil,
		"posts":     []any{},
		"literal":   "static",
	}
	assert.Equal(t, expected, result)
}

func TestBuildTableEndpointResponseGeneratorSpec(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	rows := []map[string]any{{"users__name": "Alice"}}
	template := map[string]any{
		"user_name": "{{users.name}}",
		"age":       map[string]any{"type": "int", "min": 18, "max": 80},
	}

	result := BuildTableEndpointResponse(rng, template, rows)

	m := result.(map[string]any)
	assert.Equal(t, "Alice", m["user_name"])
	assert.GreaterOrEqual(t, m["age"].(int), 18)
	assert.LessOrEqual(t, m["age"].(int), 80)
}
