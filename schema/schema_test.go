package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParamNames(t *testing.T) {
	e := Endpoint{Path: "/users/{id}/posts/{post_id}/comments/{commentId}"}
	assert.Equal(t, []string{"id", "post_id", "commentId"}, e.ParamNames())
}

func TestParamNamesNone(t *testing.T) {
	e := Endpoint{Path: "/users/static"}
	assert.Empty(t, e.ParamNames())
}

func TestValidateValidSchema(t *testing.T) {
	sch := validSchema()
	require.NoError(t, sch.Validate())
}

func TestValidateEmptySchema(t *testing.T) {
	sch := &Schema{}
	err := sch.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "must define at least one of table or endpoint")
}

func TestValidateTableNames(t *testing.T) {
	cases := []struct {
		name  string
		table Table
		want  string
	}{
		{name: "empty", table: Table{Name: " "}, want: "table name is required"},
		{name: "starts with digit", table: Table{Name: "1users"}, want: "invalid table name"},
		{name: "space in name", table: Table{Name: "my table"}, want: "invalid table name"},
		{name: "hyphen", table: Table{Name: "my-table"}, want: "invalid table name"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sch := &Schema{Tables: []Table{tc.table}}
			err := sch.Validate()
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

func TestValidateDuplicateTableNames(t *testing.T) {
	sch := &Schema{Tables: []Table{
		{Name: "users", Columns: []Column{{Name: "a", Type: "string"}}},
		{Name: "users", Columns: []Column{{Name: "b", Type: "string"}}},
	}}

	err := sch.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, `duplicate table name "users"`)
}

func TestValidateColumnNames(t *testing.T) {
	cases := []struct {
		name   string
		column Column
		want   string
	}{
		{name: "empty", column: Column{Name: "", Type: "string"}, want: "column name is required"},
		{name: "reserved id", column: Column{Name: "id", Type: "string"}, want: `column name "id" is reserved`},
		{name: "invalid", column: Column{Name: "bad name", Type: "string"}, want: "invalid column name"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{tc.column}}}}
			err := sch.Validate()
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

func TestValidateDuplicateColumnNames(t *testing.T) {
	sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{
		{Name: "a", Type: "string"},
		{Name: "a", Type: "string"},
	}}}}

	err := sch.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "duplicate column")
}

func TestValidateColumnTypes(t *testing.T) {
	sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{
		{Name: "a", Type: "string"},
		{Name: "b", Type: "int"},
		{Name: "c", Type: "float"},
		{Name: "d", Type: "bool"},
		{Name: "e", Type: "datetime"},
		{Name: "f", Type: "blob"},
	}}}}

	err := sch.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, `invalid type "blob"`)
}

func TestValidateColumnRanges(t *testing.T) {
	t.Run("min greater than max", func(t *testing.T) {
		min, max := 10.0, 5.0
		sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{
			{Name: "n", Type: "int", Min: &min, Max: &max},
		}}}}

		err := sch.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "min (10) must be <= max (5)")
	})

	t.Run("min_length greater than max_length", func(t *testing.T) {
		minLen, maxLen := 10, 5
		sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{
			{Name: "s", Type: "string", MinLength: &minLen, MaxLength: &maxLen},
		}}}}

		err := sch.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "min_length (10) must be <= max_length (5)")
	})

	t.Run("negative min_length", func(t *testing.T) {
		negative := -1
		sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{
			{Name: "s", Type: "string", MinLength: &negative},
		}}}}

		err := sch.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "min_length must be >= 0")
	})

	t.Run("negative max_length", func(t *testing.T) {
		negative := -1
		sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{
			{Name: "s", Type: "string", MaxLength: &negative},
		}}}}

		err := sch.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "max_length must be >= 0")
	})
}

func TestValidateUnknownFormat(t *testing.T) {
	sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{
		{Name: "s", Type: "string", Format: "title"},
	}}}}

	err := sch.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, `unknown format "title"`)
}

func TestValidateRegex(t *testing.T) {
	t.Run("valid regex compiles", func(t *testing.T) {
		sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{
			{Name: "s", Type: "string", Regex: "^[a-z]+$"},
		}}}}

		require.NoError(t, sch.Validate())
		require.NotNil(t, sch.Tables[0].Columns[0].RegexCompiled)
	})

	t.Run("invalid regex", func(t *testing.T) {
		sch := &Schema{Tables: []Table{{Name: "t", Columns: []Column{
			{Name: "s", Type: "string", Regex: "["},
		}}}}

		err := sch.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid regex")
	})
}

func TestValidateForeignKeys(t *testing.T) {
	t.Run("valid foreign key", func(t *testing.T) {
		sch := &Schema{Tables: []Table{
			{Name: "users", Columns: []Column{{Name: "email", Type: "string"}}},
			{Name: "posts", Columns: []Column{
				{Name: "user_id", Type: "int", ForeignKey: &ForeignKey{Table: "users"}},
			}},
		}}

		require.NoError(t, sch.Validate())
	})

	t.Run("unknown parent table", func(t *testing.T) {
		sch := &Schema{Tables: []Table{{Name: "posts", Columns: []Column{
			{Name: "user_id", Type: "int", ForeignKey: &ForeignKey{Table: "ghosts"}},
		}}}}

		err := sch.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, `foreign_key references unknown table "ghosts"`)
	})

	t.Run("unknown parent column", func(t *testing.T) {
		sch := &Schema{Tables: []Table{
			{Name: "users", Columns: []Column{{Name: "email", Type: "string"}}},
			{Name: "posts", Columns: []Column{
				{Name: "user_id", Type: "int", ForeignKey: &ForeignKey{Table: "users", Column: "nope"}},
			}},
		}}

		err := sch.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, `foreign_key references unknown column "nope"`)
	})
}

func TestValidateEndpoints(t *testing.T) {
	cases := []struct {
		name     string
		endpoint Endpoint
		want     string
	}{
		{name: "empty method", endpoint: Endpoint{Method: "", Path: "/x", Response: "a"}, want: "method is required"},
		{name: "invalid method", endpoint: Endpoint{Method: "FOO", Path: "/x", Response: "a"}, want: `invalid method "FOO"`},
		{name: "empty path", endpoint: Endpoint{Method: "GET", Path: "", Response: "a"}, want: "path is required"},
		{name: "no leading slash", endpoint: Endpoint{Method: "GET", Path: "x", Response: "a"}, want: `path "x" must start with /`},
		{name: "nil response", endpoint: Endpoint{Method: "GET", Path: "/x", Response: nil}, want: "response is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sch := &Schema{Endpoints: []Endpoint{tc.endpoint}}
			err := sch.Validate()
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

func TestValidateEndpointNormalization(t *testing.T) {
	sch := &Schema{Endpoints: []Endpoint{
		{Method: " get ", Path: "/health", Status: 0, Response: map[string]any{"ok": true}},
	}}

	require.NoError(t, sch.Validate())
	assert.Equal(t, "GET", sch.Endpoints[0].Method)
	assert.Equal(t, 200, sch.Endpoints[0].Status)
}

func TestValidateDuplicateEndpoints(t *testing.T) {
	sch := &Schema{Endpoints: []Endpoint{
		{Method: "GET", Path: "/health", Response: "a"},
		{Method: "get", Path: "/health", Response: "b"},
	}}

	err := sch.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "duplicate pattern")
}

func TestParseSchema(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "schema.json")
		content := `{
			"tables": [
				{"name": "users", "columns": [
					{"name": "email", "type": "string", "required": true}
				]}
			],
			"endpoints": [
				{"method": "GET", "path": "/health", "response": {"ok": true}}
			]
		}`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		sch, err := ParseSchema(path)
		require.NoError(t, err)
		require.Len(t, sch.Tables, 1)
		assert.Equal(t, "users", sch.Tables[0].Name)
		require.Len(t, sch.Endpoints, 1)
		assert.Equal(t, "GET", sch.Endpoints[0].Method)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := ParseSchema(filepath.Join(t.TempDir(), "does-not-exist.json"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "reading schema file")
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "schema.json")
		require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o644))

		_, err := ParseSchema(path)
		require.Error(t, err)
		assert.ErrorContains(t, err, "parsing schema JSON")
	})

	t.Run("invalid schema", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "schema.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"tables": []}`), 0o644))

		_, err := ParseSchema(path)
		require.Error(t, err)
		assert.ErrorContains(t, err, "validating schema")
	})
}

func tableEndpointSchema(ep TableEndpoint) *Schema {
	return &Schema{
		Tables: []Table{
			{
				Name: "users",
				Columns: []Column{
					{Name: "name", Type: "string"},
					{Name: "email", Type: "string"},
				},
			},
			{
				Name: "posts",
				Columns: []Column{
					{Name: "title", Type: "string"},
					{Name: "user_id", Type: "int", ForeignKey: &ForeignKey{Table: "users"}},
				},
			},
		},
		TableEndpoints: []TableEndpoint{ep},
	}
}

func TestValidateTableEndpointsValid(t *testing.T) {
	ep := TableEndpoint{
		Method: "GET",
		Path:   "/users/{id}/posts",
		Tables: []string{"users", "posts"},
		Where:  []string{"users.id = {{path.id}}", "posts.status = 'published'"},
		Response: map[string]any{
			"user_name": "{{users.name}}",
			"posts": map[string]any{
				"type": "array",
				"items": map[string]any{
					"title": "{{posts.title}}",
				},
			},
		},
	}
	require.NoError(t, tableEndpointSchema(ep).Validate())
}

func TestValidateTableEndpointsInvalidMethod(t *testing.T) {
	ep := TableEndpoint{
		Method:   "POST",
		Path:     "/users/{id}/posts",
		Tables:   []string{"users", "posts"},
		Response: map[string]any{"x": "{{users.name}}"},
	}
	err := tableEndpointSchema(ep).Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "only support GET")
}

func TestValidateTableEndpointsMissingTables(t *testing.T) {
	ep := TableEndpoint{
		Method:   "GET",
		Path:     "/x",
		Tables:   []string{"ghosts"},
		Response: map[string]any{"x": "{{users.name}}"},
	}
	err := tableEndpointSchema(ep).Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, `unknown table "ghosts"`)
}

func TestValidateTableEndpointsDuplicateTables(t *testing.T) {
	ep := TableEndpoint{
		Method:   "GET",
		Path:     "/x",
		Tables:   []string{"users", "users"},
		Response: map[string]any{"x": "{{users.name}}"},
	}
	err := tableEndpointSchema(ep).Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, `duplicate table "users"`)
}

func TestValidateTableEndpointsInvalidJoinRef(t *testing.T) {
	ep := TableEndpoint{
		Method: "GET",
		Path:   "/x",
		Tables: []string{"users", "posts"},
		Joins: []Join{
			{Type: "INNER", On: JoinCondition{Local: "users.id", Foreign: "posts.notacolumn"}},
		},
		Response: map[string]any{"x": "{{users.name}}"},
	}
	err := tableEndpointSchema(ep).Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, `unknown column "notacolumn"`)
}

func TestValidateTableEndpointsUnknownResponseColumn(t *testing.T) {
	ep := TableEndpoint{
		Method:   "GET",
		Path:     "/x",
		Tables:   []string{"users", "posts"},
		Response: map[string]any{"x": "{{users.nonexistent}}"},
	}
	err := tableEndpointSchema(ep).Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, `unknown column "nonexistent"`)
}

func TestValidateTableEndpointsUnknownWhereTable(t *testing.T) {
	ep := TableEndpoint{
		Method:   "GET",
		Path:     "/x",
		Tables:   []string{"users", "posts"},
		Where:    []string{"ghosts.id = 1"},
		Response: map[string]any{"x": "{{users.name}}"},
	}
	err := tableEndpointSchema(ep).Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, `unknown table "ghosts"`)
}

func TestValidateTableEndpointsDuplicatePattern(t *testing.T) {
	ep := TableEndpoint{
		Method:   "GET",
		Path:     "/users/{id}/posts",
		Tables:   []string{"users", "posts"},
		Response: map[string]any{"x": "{{users.name}}"},
	}
	sch := tableEndpointSchema(ep)
	sch.Endpoints = []Endpoint{{Method: "GET", Path: "/users/{id}/posts", Response: "x"}}
	err := sch.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "duplicate pattern")
}

func TestValidateTableEndpointsShadowsCRUD(t *testing.T) {
	ep := TableEndpoint{
		Method:   "GET",
		Path:     "/users",
		Tables:   []string{"users"},
		Response: map[string]any{"x": "{{users.name}}"},
	}
	err := tableEndpointSchema(ep).Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "shadows a CRUD route")
}

func TestValidateTableEndpointsStatusDefault(t *testing.T) {
	ep := TableEndpoint{
		Method:   "GET",
		Path:     "/x",
		Tables:   []string{"users"},
		Response: map[string]any{"x": "{{users.name}}"},
	}
	sch := tableEndpointSchema(ep)
	require.NoError(t, sch.Validate())
	assert.Equal(t, 200, sch.TableEndpoints[0].Status)
}

func TestValidateTableEndpointsJoinOnReservedId(t *testing.T) {
	ep := TableEndpoint{
		Method: "GET",
		Path:   "/users/{id}/posts",
		Tables: []string{"users", "posts"},
		Joins: []Join{
			{Type: "INNER", On: JoinCondition{Local: "users.id", Foreign: "posts.user_id"}},
		},
		Response: map[string]any{"x": "{{users.id}}", "y": "{{posts.title}}"},
	}
	err := tableEndpointSchema(ep).Validate()
	require.NoError(t, err)
}

func validSchema() *Schema {
	return &Schema{
		Tables: []Table{
			{
				Name: "users",
				Columns: []Column{
					{Name: "name", Type: "string", Required: true},
					{Name: "email", Type: "string", Unique: true, Regex: "^[^@]+@[^@]+$"},
					{Name: "age", Type: "int", Min: ptr(18.0), Max: ptr(99.0)},
				},
			},
			{
				Name: "posts",
				Columns: []Column{
					{Name: "title", Type: "string", Format: "uuid"},
					{Name: "user_id", Type: "int", ForeignKey: &ForeignKey{Table: "users"}},
				},
			},
		},
		Endpoints: []Endpoint{
			{Method: "GET", Path: "/health", Status: 200, Response: map[string]any{"ok": true}},
		},
	}
}

func ptr[T any](v T) *T {
	return &v
}
