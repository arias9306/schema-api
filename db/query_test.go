package db

import (
	"testing"

	"github.com/arias9306/schema-api/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTableMap() map[string]schema.Table {
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
			{Name: "user_id", Type: "int", ForeignKey: &schema.ForeignKey{Table: "users"}},
		},
	}
	return map[string]schema.Table{
		"users": users,
		"posts": posts,
	}
}

func TestBuildSelectColumns(t *testing.T) {
	tableMap := testTableMap()
	response := map[string]any{
		"user_name": "{{users.name}}",
		"posts": map[string]any{
			"type": "array",
			"items": map[string]any{
				"title": "{{posts.title}}",
				"body":  "{{posts.body}}",
			},
		},
	}

	cols := BuildSelectColumns(response, []string{"users", "posts"}, tableMap)

	assert.Contains(t, cols, `"users"."id" AS "users__id"`)
	assert.Contains(t, cols, `"users"."name" AS "users__name"`)
	assert.Contains(t, cols, `"posts"."id" AS "posts__id"`)
	assert.Contains(t, cols, `"posts"."title" AS "posts__title"`)
	assert.Contains(t, cols, `"posts"."body" AS "posts__body"`)
	assert.NotContains(t, cols, "email")
}

func TestInferJoins(t *testing.T) {
	tableMap := testTableMap()
	joins, err := InferJoins([]string{"users", "posts"}, tableMap)
	require.NoError(t, err)
	require.Len(t, joins, 1)
	assert.Equal(t, "INNER", joins[0].Type)
	assert.Equal(t, "users.id", joins[0].On.Local)
	assert.Equal(t, "posts.user_id", joins[0].On.Foreign)
}

func TestInferJoinsUnjoinable(t *testing.T) {
	tableMap := testTableMap()
	_, err := InferJoins([]string{"posts", "users"}, tableMap)
	require.NoError(t, err)
}

func TestBuildJoinClause(t *testing.T) {
	joins := []schema.Join{
		{Type: "INNER", On: schema.JoinCondition{Local: "users.id", Foreign: "posts.user_id"}},
	}
	clause := BuildJoinClause(joins)
	assert.Equal(t, ` INNER JOIN "posts" ON "users"."id" = "posts"."user_id"`, clause)
}

func TestInterpolateWhere(t *testing.T) {
	ctx := map[string]string{"path.id": "42"}
	clause, params, err := InterpolateWhere(
		[]string{"users.id = {{path.id}}", "posts.status = 'published'"},
		ctx,
	)
	require.NoError(t, err)
	assert.Equal(t, `users.id = ? AND posts.status = 'published'`, clause)
	assert.Equal(t, []any{"42"}, params)
}

func TestBuildQuery(t *testing.T) {
	tableMap := testTableMap()
	limit := 10
	ep := schema.TableEndpoint{
		Method:  "GET",
		Path:    "/users/{id}/posts",
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

	query, params, err := BuildQuery(ep, tableMap, map[string]string{"path.id": "42"})
	require.NoError(t, err)

	assert.Contains(t, query, `SELECT "users"."id" AS "users__id", "users"."name" AS "users__name", "posts"."id" AS "posts__id", "posts"."title" AS "posts__title"`)
	assert.Contains(t, query, `FROM "users"`)
	assert.Contains(t, query, `INNER JOIN "posts" ON "users"."id" = "posts"."user_id"`)
	assert.Contains(t, query, `WHERE users.id = ? AND posts.status = 'published'`)
	assert.Contains(t, query, "ORDER BY posts.id DESC")
	assert.Contains(t, query, "LIMIT ?")
	assert.Equal(t, []any{"42", int(10)}, params)
}

func TestBuildQueryExplicitJoins(t *testing.T) {
	tableMap := testTableMap()
	ep := schema.TableEndpoint{
		Method: "GET",
		Path:   "/users/{id}/posts",
		Tables: []string{"users", "posts"},
		Joins: []schema.Join{
			{Type: "LEFT", On: schema.JoinCondition{Local: "users.id", Foreign: "posts.user_id"}},
		},
		Response: map[string]any{"name": "{{users.name}}"},
	}

	query, _, err := BuildQuery(ep, tableMap, map[string]string{})
	require.NoError(t, err)
	assert.Contains(t, query, `LEFT JOIN "posts" ON "users"."id" = "posts"."user_id"`)
}
