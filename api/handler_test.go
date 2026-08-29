package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqlite3 "modernc.org/sqlite/lib"
)

func newTestHandler(t *testing.T) (*Handler, *http.ServeMux) {
	t.Helper()
	database, err := db.InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	sch := schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "name", Type: "string", Required: true},
					{Name: "email", Type: "string", Unique: true},
					{Name: "age", Type: "int"},
					{Name: "active", Type: "bool"},
				},
			},
			{
				Name: "posts",
				Columns: []schema.Column{
					{Name: "title", Type: "string", Required: true},
					{Name: "user_id", Type: "int", ForeignKey: &schema.ForeignKey{Table: "users"}},
				},
			},
		},
	}

	for _, table := range sch.Tables {
		require.NoError(t, db.CreateTable(database, table))
	}

	h := NewAPIHandler(database, &sch)
	mux := http.NewServeMux()
	h.Register(mux)
	return h, mux
}

func doRequest(t *testing.T, mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func seedUser(t *testing.T, h *Handler, name, email string) int64 {
	t.Helper()
	id, err := db.Insert(h.db, h.tables["users"], map[string]any{
		"name": name, "email": email, "age": int64(30), "active": true,
	})
	require.NoError(t, err)
	return id
}

func decodeRows(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(body, &rows))
	return rows
}

func decodeRow(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var row map[string]any
	require.NoError(t, json.Unmarshal(body, &row))
	return row
}

func TestList(t *testing.T) {
	h, mux := newTestHandler(t)
	seedUser(t, h, "Alice", "alice@example.com")
	seedUser(t, h, "Bob", "bob@example.com")
	seedUser(t, h, "Carol", "carol@example.com")

	t.Run("returns rows and headers", func(t *testing.T) {
		rec := doRequest(t, mux, "GET", "/users", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "3", rec.Header().Get("X-Total-Count"))
		assert.Equal(t, "1", rec.Header().Get("X-Page"))
		assert.Equal(t, "20", rec.Header().Get("X-Limit"))

		rows := decodeRows(t, rec.Body.Bytes())
		require.Len(t, rows, 3)
		assert.Equal(t, "Alice", rows[0]["name"])
		assert.Equal(t, true, rows[0]["active"])
	})

	t.Run("pagination", func(t *testing.T) {
		rec := doRequest(t, mux, "GET", "/users?page=2&limit=1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "2", rec.Header().Get("X-Page"))
		assert.Equal(t, "1", rec.Header().Get("X-Limit"))

		rows := decodeRows(t, rec.Body.Bytes())
		require.Len(t, rows, 1)
		assert.Equal(t, "Bob", rows[0]["name"])
	})

	t.Run("limit clamped to 20", func(t *testing.T) {
		rec := doRequest(t, mux, "GET", "/users?limit=500", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "20", rec.Header().Get("X-Limit"))
	})

	t.Run("sort and order", func(t *testing.T) {
		rec := doRequest(t, mux, "GET", "/users?sort=name&order=desc", "")
		require.Equal(t, http.StatusOK, rec.Code)

		rows := decodeRows(t, rec.Body.Bytes())
		require.Len(t, rows, 3)
		assert.Equal(t, "Carol", rows[0]["name"])
		assert.Equal(t, "Alice", rows[2]["name"])
	})

	t.Run("unknown table", func(t *testing.T) {
		rec := doRequest(t, mux, "GET", "/nope", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "table not found")
	})
}

func TestGet(t *testing.T) {
	h, mux := newTestHandler(t)
	id := seedUser(t, h, "Alice", "alice@example.com")
	idStr := strconv.FormatInt(id, 10)

	t.Run("found", func(t *testing.T) {
		rec := doRequest(t, mux, "GET", "/users/"+idStr, "")
		require.Equal(t, http.StatusOK, rec.Code)

		row := decodeRow(t, rec.Body.Bytes())
		assert.Equal(t, "Alice", row["name"])
		assert.Equal(t, "alice@example.com", row["email"])
		assert.Equal(t, true, row["active"])
	})

	t.Run("not found", func(t *testing.T) {
		rec := doRequest(t, mux, "GET", "/users/999", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "row not found")
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := doRequest(t, mux, "GET", "/users/abc", "")
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid id")
	})

	t.Run("unknown table", func(t *testing.T) {
		rec := doRequest(t, mux, "GET", "/nope/1", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestCreate(t *testing.T) {
	_, mux := newTestHandler(t)

	t.Run("created", func(t *testing.T) {
		rec := doRequest(t, mux, "POST", "/users", `{"name":"Alice","email":"alice@example.com","age":30}`)
		require.Equal(t, http.StatusCreated, rec.Code)

		row := decodeRow(t, rec.Body.Bytes())
		assert.Equal(t, "Alice", row["name"])
		id, ok := row["id"].(float64)
		require.True(t, ok)
		assert.Greater(t, id, 0.0)
	})

	t.Run("invalid json", func(t *testing.T) {
		rec := doRequest(t, mux, "POST", "/users", `{not json`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid JSON")
	})

	t.Run("body too large", func(t *testing.T) {
		big := `{"name":"` + strings.Repeat("a", maxBodyBytes+100) + `"}`
		rec := doRequest(t, mux, "POST", "/users", big)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		rec := doRequest(t, mux, "POST", "/users", `{"email":"alice@example.com"}`)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "name is required")
	})

	t.Run("unique conflict", func(t *testing.T) {
		doRequest(t, mux, "POST", "/users", `{"name":"A","email":"dup@example.com"}`)
		rec := doRequest(t, mux, "POST", "/users", `{"name":"B","email":"dup@example.com"}`)
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "unique constraint violation")
	})

	t.Run("foreign key violation", func(t *testing.T) {
		rec := doRequest(t, mux, "POST", "/posts", `{"title":"post","user_id":999}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "foreign key violation")
	})

	t.Run("unknown table", func(t *testing.T) {
		rec := doRequest(t, mux, "POST", "/nope", `{}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestUpdate(t *testing.T) {
	h, mux := newTestHandler(t)
	id := seedUser(t, h, "Alice", "alice@example.com")
	idStr := strconv.FormatInt(id, 10)

	t.Run("updated", func(t *testing.T) {
		rec := doRequest(t, mux, "PUT", "/users/"+idStr, `{"age":31}`)
		require.Equal(t, http.StatusOK, rec.Code)

		row := decodeRow(t, rec.Body.Bytes())
		assert.Equal(t, float64(31), row["age"])
		assert.Equal(t, "Alice", row["name"])
	})

	t.Run("not found", func(t *testing.T) {
		rec := doRequest(t, mux, "PUT", "/users/999", `{"age":31}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := doRequest(t, mux, "PUT", "/users/abc", `{}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		rec := doRequest(t, mux, "PUT", "/users/"+idStr, `{"age":"old"}`)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "age: must be a number")
	})

	t.Run("unique conflict", func(t *testing.T) {
		seedUser(t, h, "Bob", "bob@example.com")
		rec := doRequest(t, mux, "PUT", "/users/"+idStr, `{"email":"bob@example.com"}`)
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "unique constraint violation")
	})

	t.Run("unknown table", func(t *testing.T) {
		rec := doRequest(t, mux, "PUT", "/nope/1", `{}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestDelete(t *testing.T) {
	h, mux := newTestHandler(t)

	t.Run("deleted", func(t *testing.T) {
		id := seedUser(t, h, "Alice", "alice@example.com")
		rec := doRequest(t, mux, "DELETE", "/users/"+strconv.FormatInt(id, 10), "")
		require.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("not found", func(t *testing.T) {
		rec := doRequest(t, mux, "DELETE", "/users/999", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "row not found")
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := doRequest(t, mux, "DELETE", "/users/abc", "")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unknown table", func(t *testing.T) {
		rec := doRequest(t, mux, "DELETE", "/nope/1", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("referenced by other rows", func(t *testing.T) {
		userID := seedUser(t, h, "Carol", "carol@example.com")
		_, err := db.Insert(h.db, h.tables["posts"], map[string]any{"title": "t", "user_id": userID})
		require.NoError(t, err)

		rec := doRequest(t, mux, "DELETE", "/users/"+strconv.FormatInt(userID, 10), "")
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "cannot delete: referenced by other rows")
	})
}

func TestHandlerInternalErrors(t *testing.T) {
	database, err := db.InitDB(":memory:")
	require.NoError(t, err)

	sch := schema.Schema{Tables: []schema.Table{
		{Name: "users", Columns: []schema.Column{{Name: "name", Type: "string"}}},
	}}
	require.NoError(t, db.CreateTable(database, sch.Tables[0]))

	h := NewAPIHandler(database, &sch)
	mux := http.NewServeMux()
	h.Register(mux)

	require.NoError(t, database.Close())

	cases := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "list", method: "GET", target: "/users"},
		{name: "get", method: "GET", target: "/users/1"},
		{name: "create", method: "POST", target: "/users", body: `{"name":"x"}`},
		{name: "update", method: "PUT", target: "/users/1", body: `{"name":"x"}`},
		{name: "delete", method: "DELETE", target: "/users/1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(t, mux, tc.method, tc.target, tc.body)
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	}
}

func TestConstraintCode(t *testing.T) {
	database, err := db.InitDB(":memory:")
	require.NoError(t, err)
	defer database.Close()

	sch := schema.Schema{Tables: []schema.Table{
		{Name: "users", Columns: []schema.Column{{Name: "email", Type: "string", Unique: true}}},
		{Name: "posts", Columns: []schema.Column{
			{Name: "user_id", Type: "int", ForeignKey: &schema.ForeignKey{Table: "users"}},
		}},
	}}
	for _, table := range sch.Tables {
		require.NoError(t, db.CreateTable(database, table))
	}

	t.Run("unique constraint", func(t *testing.T) {
		_, err := db.Insert(database, sch.Tables[0], map[string]any{"email": "a@example.com"})
		require.NoError(t, err)
		_, err = db.Insert(database, sch.Tables[0], map[string]any{"email": "a@example.com"})
		require.Error(t, err)

		code := constraintCode(err)
		require.NotNil(t, code)
		assert.Equal(t, sqlite3.SQLITE_CONSTRAINT_UNIQUE, *code)
	})

	t.Run("foreign key constraint", func(t *testing.T) {
		_, err := db.Insert(database, sch.Tables[1], map[string]any{"user_id": int64(999)})
		require.Error(t, err)

		code := constraintCode(err)
		require.NotNil(t, code)
		assert.Equal(t, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY, *code)
	})

	t.Run("non constraint error", func(t *testing.T) {
		assert.Nil(t, constraintCode(errors.New("something else")))
	})
}

func TestWriteConstraintError(t *testing.T) {
	cases := []struct {
		code   int
		status int
		msg    string
	}{
		{sqlite3.SQLITE_CONSTRAINT_UNIQUE, http.StatusConflict, "unique constraint violation"},
		{sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, http.StatusConflict, "unique constraint violation"},
		{sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY, http.StatusBadRequest, "foreign key violation"},
		{sqlite3.SQLITE_CONSTRAINT_CHECK, http.StatusBadRequest, "constraint violation"},
	}

	for _, tc := range cases {
		rec := httptest.NewRecorder()
		writeConstraintError(rec, tc.code)
		assert.Equal(t, tc.status, rec.Code, "code %d", tc.code)
		assert.Contains(t, rec.Body.String(), tc.msg, "code %d", tc.code)
	}
}

func TestTableEndpointHandlerRequest(t *testing.T) {
	database, err := db.InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

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
	handler := NewAPIHandler(database, sch)
	require.NoError(t, handler.RegisterTableEndpoints(mux))

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
	t.Cleanup(func() { database.Close() })

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
	handler := NewAPIHandler(database, sch)
	require.NoError(t, handler.RegisterTableEndpoints(mux))

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
