package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableEndpointHandlerRequest(t *testing.T) {
	database, err := db.InitDB(":memory:")
	require.NoError(t, err)

	users := schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{Name: "name", Type: "string"},
			{Name: "email", Type: "string"},
		},
	}
	posts := schema.Table{
		Name: "posts",
		Columns: []schema.Column{
			{Name: "title", Type: "string"},
			{Name: "body", Type: "string"},
			{Name: "status", Type: "string"},
			{Name: "user_id", Type: "int", ForeignKey: &schema.ForeignKey{Table: "users"}},
		},
	}

	require.NoError(t, db.CreateTable(database, users))
	require.NoError(t, db.CreateTable(database, posts))

	userID, err := db.Insert(database, users, map[string]any{"name": "Alice", "email": "alice@example.com"})
	require.NoError(t, err)
	_, err = db.Insert(database, posts, map[string]any{"title": "First Post", "body": "first", "status": "published", "user_id": userID})
	require.NoError(t, err)
	_, err = db.Insert(database, posts, map[string]any{"title": "Second Post", "body": "second", "status": "published", "user_id": userID})
	require.NoError(t, err)
	_, err = db.Insert(database, posts, map[string]any{"title": "Draft", "body": "draft", "status": "draft", "user_id": userID})
	require.NoError(t, err)

	limit := 10
	sch := &schema.Schema{
		Tables: []schema.Table{users, posts},
		TableEndpoints: []schema.TableEndpoint{
			{
				Method:  "GET",
				Path:    "/users/{id}/posts",
				Tables:  []string{"users", "posts"},
				Where:   []string{"users.id = {{path.id}}", "posts.status = 'published'"},
				OrderBy: "posts.id DESC",
				Limit:   &limit,
				Headers: map[string]string{"X-Source": "database"},
				Response: map[string]any{
					"user_name":  "{{users.name}}",
					"user_email": "{{users.email}}",
					"posts": map[string]any{
						"type": "array",
						"items": map[string]any{
							"title": "{{posts.title}}",
							"body":  "{{posts.body}}",
						},
					},
				},
			},
		},
	}

	mux := http.NewServeMux()
	handler := NewTableEndpointHandler(database, sch)
	require.NoError(t, handler.Register(mux))

	req := httptest.NewRequest(http.MethodGet, "/users/1/posts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "database", rec.Header().Get("X-Source"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	assert.Equal(t, "Alice", body["user_name"])
	assert.Equal(t, "alice@example.com", body["user_email"])

	postsList, ok := body["posts"].([]any)
	require.True(t, ok)
	require.Len(t, postsList, 2, "draft post should be excluded")

	first := postsList[0].(map[string]any)
	assert.Equal(t, "Second Post", first["title"])
}

func TestTableEndpointHandlerEmpty(t *testing.T) {
	database, err := db.InitDB(":memory:")
	require.NoError(t, err)

	users := schema.Table{
		Name:    "users",
		Columns: []schema.Column{{Name: "name", Type: "string"}},
	}
	posts := schema.Table{
		Name: "posts",
		Columns: []schema.Column{
			{Name: "title", Type: "string"},
			{Name: "user_id", Type: "int", ForeignKey: &schema.ForeignKey{Table: "users"}},
		},
	}
	require.NoError(t, db.CreateTable(database, users))
	require.NoError(t, db.CreateTable(database, posts))

	sch := &schema.Schema{
		Tables: []schema.Table{users, posts},
		TableEndpoints: []schema.TableEndpoint{
			{
				Method: "GET",
				Path:   "/users/{id}/posts",
				Tables: []string{"users", "posts"},
				Where:  []string{"users.id = {{path.id}}"},
				Response: map[string]any{
					"user_name": "{{users.name}}",
					"posts": map[string]any{
						"type":  "array",
						"items": map[string]any{"title": "{{posts.title}}"},
					},
				},
			},
		},
	}

	mux := http.NewServeMux()
	handler := NewTableEndpointHandler(database, sch)
	require.NoError(t, handler.Register(mux))

	req := httptest.NewRequest(http.MethodGet, "/users/999/posts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d body=%s", rec.Code, rec.Body.String())
	}
	require.Equal(t, http.StatusOK, rec.Code)

	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"posts":[],"user_name":null}`, strings.TrimSpace(string(body)))
}
