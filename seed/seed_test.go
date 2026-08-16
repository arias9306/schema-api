package seed

import (
	"database/sql"
	"fmt"
	"math/rand"
	"testing"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tableWithFK(name string, fks ...schema.ForeignKey) schema.Table {
	table := schema.Table{Name: name}
	for i, fk := range fks {
		table.Columns = append(table.Columns, schema.Column{
			Name:       fmt.Sprintf("ref%d", i),
			Type:       "int",
			ForeignKey: &fk,
		})
	}
	return table
}

func indexOf(tables []schema.Table, name string) int {
	for i, table := range tables {
		if table.Name == name {
			return i
		}
	}
	return -1
}

func TestTopologicalSort(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result, err := topologicalSort(nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("chain", func(t *testing.T) {
		tables := []schema.Table{
			tableWithFK("c", schema.ForeignKey{Table: "b"}),
			tableWithFK("a"),
			tableWithFK("b", schema.ForeignKey{Table: "a"}),
		}

		result, err := topologicalSort(tables)
		require.NoError(t, err)
		require.Len(t, result, 3)
		assert.Less(t, indexOf(result, "a"), indexOf(result, "b"))
		assert.Less(t, indexOf(result, "b"), indexOf(result, "c"))
	})

	t.Run("diamond", func(t *testing.T) {
		tables := []schema.Table{
			tableWithFK("a"),
			tableWithFK("b", schema.ForeignKey{Table: "a"}),
			tableWithFK("c", schema.ForeignKey{Table: "a"}),
			tableWithFK("d", schema.ForeignKey{Table: "b"}, schema.ForeignKey{Table: "c"}),
		}

		result, err := topologicalSort(tables)
		require.NoError(t, err)
		require.Len(t, result, 4)
		assert.Less(t, indexOf(result, "a"), indexOf(result, "b"))
		assert.Less(t, indexOf(result, "a"), indexOf(result, "c"))
		assert.Less(t, indexOf(result, "b"), indexOf(result, "d"))
		assert.Less(t, indexOf(result, "c"), indexOf(result, "d"))
	})

	t.Run("independent tables keep all", func(t *testing.T) {
		tables := []schema.Table{
			tableWithFK("x"),
			tableWithFK("y"),
		}

		result, err := topologicalSort(tables)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("circular dependency", func(t *testing.T) {
		tables := []schema.Table{
			tableWithFK("a", schema.ForeignKey{Table: "b"}),
			tableWithFK("b", schema.ForeignKey{Table: "a"}),
		}

		_, err := topologicalSort(tables)
		require.Error(t, err)
		assert.ErrorContains(t, err, "circular dependency detected")
	})

	t.Run("self reference", func(t *testing.T) {
		tables := []schema.Table{
			tableWithFK("a", schema.ForeignKey{Table: "a"}),
		}

		_, err := topologicalSort(tables)
		require.Error(t, err)
		assert.ErrorContains(t, err, "circular dependency detected")
	})
}

func TestSeed(t *testing.T) {
	database, err := db.InitDB(":memory:")
	require.NoError(t, err)
	defer database.Close()

	sch := &schema.Schema{Tables: []schema.Table{
		{Name: "users", Columns: []schema.Column{
			{Name: "name", Type: "string", Required: true},
			{Name: "email", Type: "string", Unique: true},
		}},
		{Name: "posts", Columns: []schema.Column{
			{Name: "title", Type: "string", Required: true},
			{Name: "user_id", Type: "int", ForeignKey: &schema.ForeignKey{Table: "users"}},
		}},
		{Name: "comments", Columns: []schema.Column{
			{Name: "body", Type: "string"},
			{Name: "post_id", Type: "int", ForeignKey: &schema.ForeignKey{Table: "posts"}},
		}},
	}}

	for _, table := range sch.Tables {
		require.NoError(t, db.CreateTable(database, table))
	}

	const rowsPerTable = 25
	require.NoError(t, Seed(database, sch, rowsPerTable))

	for _, table := range sch.Tables {
		var count int
		require.NoError(t, database.QueryRow("SELECT COUNT(1) FROM \""+table.Name+"\"").Scan(&count))
		assert.Equal(t, rowsPerTable, count, "row count for %s", table.Name)
	}

	emails := map[string]bool{}
	rows, err := database.Query("SELECT email FROM users")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var email string
		require.NoError(t, rows.Scan(&email))
		assert.False(t, emails[email], "duplicate email %q", email)
		emails[email] = true
	}

	userIDs := collectIDs(t, database, "users")
	postIDs := collectIDs(t, database, "posts")
	assertFKValues(t, database, "posts", "user_id", userIDs)
	assertFKValues(t, database, "comments", "post_id", postIDs)
}

func TestSeedZeroRows(t *testing.T) {
	database, err := db.InitDB(":memory:")
	require.NoError(t, err)
	defer database.Close()

	sch := &schema.Schema{Tables: []schema.Table{
		{Name: "users", Columns: []schema.Column{{Name: "name", Type: "string", Required: true}}},
	}}
	require.NoError(t, db.CreateTable(database, sch.Tables[0]))

	require.NoError(t, Seed(database, sch, 0))

	var count int
	require.NoError(t, database.QueryRow("SELECT COUNT(1) FROM users").Scan(&count))
	assert.Zero(t, count)
}

func collectIDs(t *testing.T, database *sql.DB, table string) map[int64]bool {
	t.Helper()
	ids := map[int64]bool{}
	rows, err := database.Query("SELECT id FROM \"" + table + "\"")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var id int64
		require.NoError(t, rows.Scan(&id))
		ids[id] = true
	}
	return ids
}

func assertFKValues(t *testing.T, database *sql.DB, table, column string, valid map[int64]bool) {
	t.Helper()
	rows, err := database.Query("SELECT " + column + " FROM \"" + table + "\"")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var v int64
		require.NoError(t, rows.Scan(&v))
		assert.True(t, valid[v], "%s.%s references missing id %d", table, column, v)
	}
}

func TestGenerateValue(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	used := map[string]bool{}

	for i := 0; i < 50; i++ {
		v, err := generateValue(rng, schema.Column{Name: "email", Type: "string", Unique: true}, used)
		require.NoError(t, err)
		s, ok := v.(string)
		require.True(t, ok)
		assert.Contains(t, s, "@")
		assert.False(t, used[s], "generated duplicate unique value")
		used[s] = true
	}
}

func TestGenerateValueUnsupportedType(t *testing.T) {
	_, err := generateValue(rand.New(rand.NewSource(1)), schema.Column{Name: "x", Type: "blob"}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, `unsupported type "blob"`)
}

func TestGenerateValueUniqueExhaustion(t *testing.T) {
	min, max := 1.0, 1.0
	col := schema.Column{Name: "n", Type: "int", Unique: true, Min: &min, Max: &max}
	used := map[string]bool{"1": true}

	_, err := generateValue(rand.New(rand.NewSource(1)), col, used)
	require.Error(t, err)
	assert.ErrorContains(t, err, "could not generate a unique value")
}

func TestColumnToSpec(t *testing.T) {
	min, max := 1.0, 5.0
	minLen, maxLen := 2, 10

	spec := columnToSpec(schema.Column{
		Name: "email", Type: "string",
		Min: &min, Max: &max, MinLength: &minLen, MaxLength: &maxLen,
		Regex: "^x$", Format: "EMAIL", Default: "d",
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

func TestResolveFormat(t *testing.T) {
	assert.Equal(t, "email", resolveFormat(schema.Column{Name: "email", Type: "string"}))
	assert.Equal(t, "", resolveFormat(schema.Column{Name: "age", Type: "int"}))
	assert.Equal(t, "uuid", resolveFormat(schema.Column{Name: "whatever", Type: "string", Format: "UUID"}))
}
