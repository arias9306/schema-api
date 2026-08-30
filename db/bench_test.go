package db

import (
	"database/sql"
	"math/rand"
	"strconv"
	"testing"

	"github.com/arias9306/schema-api/schema"
)

func benchSeededDB(b *testing.B, rows int) (*sql.DB, schema.Table) {
	b.Helper()
	database := openTestDB(b)
	sch := createTestTables(b, database)
	users := sch.Tables[0]

	for i := range rows {
		_, err := Insert(database, users, map[string]any{
			"name":   "user",
			"email":  "user" + strconv.Itoa(i) + "@example.com",
			"age":    int64(30),
			"active": true,
			"score":  9.5,
			"joined": "2026-01-02T03:04:05Z",
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	return database, users
}

func BenchmarkSelectAll(b *testing.B) {
	database, users := benchSeededDB(b, 100)
	b.ResetTimer()

	for b.Loop() {
		_, _, err := SelectAll(database, users, 1, 20, "", "")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelectByID(b *testing.B) {
	database, users := benchSeededDB(b, 100)
	b.ResetTimer()

	for b.Loop() {
		_, err := SelectByID(database, users, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertValue(b *testing.B) {
	users := schema.Table{Columns: []schema.Column{
		{Name: "active", Type: "bool"},
		{Name: "name", Type: "string"},
	}}
	boolCols := BoolColumnSet(users)

	b.ResetTimer()
	for b.Loop() {
		_ = ConvertValue("active", int64(1), boolCols)
	}
}

func BenchmarkBuildQuery(b *testing.B) {
	tableMap := testTableMap()
	limit := 10
	ep := schema.TableEndpoint{
		Tables:  []string{"users", "posts"},
		Where:   []string{"users.id = {{path.id}}", "posts.status = 'published'"},
		OrderBy: "posts.id DESC",
		Limit:   &limit,
		Response: map[string]any{
			"user_name": "{{users.name}}",
			"posts": map[string]any{
				"type":  "array",
				"items": map[string]any{"title": "{{posts.title}}"},
			},
		},
	}
	ctx := map[string]string{"path.id": "42"}

	b.ResetTimer()
	for b.Loop() {
		if _, _, err := BuildQuery(ep, tableMap, ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderTemplate(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	tpl := map[string]any{
		"id": "{{users.id}}",
		"posts": map[string]any{
			"type":  "array",
			"items": map[string]any{"title": "{{posts.title}}"},
		},
		"meta": map[string]any{"count": "{{posts.body}}"},
	}
	rows := []map[string]any{
		{"users__id": int64(1), "posts__id": int64(10), "posts__title": "hello", "posts__body": "world"},
		{"users__id": int64(1), "posts__id": int64(11), "posts__title": "two", "posts__body": "again"},
	}

	b.ResetTimer()
	for b.Loop() {
		_ = BuildTableEndpointResponse(rng, tpl, rows)
	}
}
