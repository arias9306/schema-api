package db

import (
	"database/sql"
	"testing"

	"github.com/arias9306/schema-api/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t testing.TB) *sql.DB {
	t.Helper()
	database, err := InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })
	return database
}

func createTestTables(t testing.TB, database *sql.DB) schema.Schema {
	t.Helper()
	sch := schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "name", Type: "string", Required: true},
					{Name: "email", Type: "string", Unique: true},
					{Name: "age", Type: "int"},
					{Name: "active", Type: "bool"},
					{Name: "score", Type: "float"},
					{Name: "joined", Type: "datetime"},
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
		require.NoError(t, CreateTable(database, table), "creating table %s", table.Name)
	}

	return sch
}

func insertUser(t *testing.T, database *sql.DB, table schema.Table, name, email string) int64 {
	t.Helper()
	id, err := Insert(database, table, map[string]any{
		"name":   name,
		"email":  email,
		"age":    int64(30),
		"active": true,
		"score":  9.5,
		"joined": "2026-01-02T03:04:05Z",
	})
	require.NoError(t, err)
	return id
}

func TestInitDB(t *testing.T) {
	database := openTestDB(t)
	require.NoError(t, database.Ping())
}

func TestCreateTable(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)

	t.Run("idempotent", func(t *testing.T) {
		require.NoError(t, CreateTable(database, sch.Tables[0]))
	})

	t.Run("id primary key autoincrements", func(t *testing.T) {
		users := sch.Tables[0]
		id1 := insertUser(t, database, users, "Alice", "alice@example.com")
		id2 := insertUser(t, database, users, "Bob", "bob@example.com")
		assert.Greater(t, id2, id1)
	})

	t.Run("foreign key to named column", func(t *testing.T) {
		parent := schema.Table{
			Name:    "teams",
			Columns: []schema.Column{{Name: "code", Type: "string", Unique: true}},
		}
		child := schema.Table{
			Name: "members",
			Columns: []schema.Column{
				{Name: "team_code", Type: "string", ForeignKey: &schema.ForeignKey{Table: "teams", Column: "code"}},
			},
		}

		require.NoError(t, CreateTable(database, parent))
		require.NoError(t, CreateTable(database, child))

		_, err := Insert(database, parent, map[string]any{"code": "alpha"})
		require.NoError(t, err)

		_, err = Insert(database, child, map[string]any{"team_code": "alpha"})
		require.NoError(t, err)

		_, err = Insert(database, child, map[string]any{"team_code": "ghost"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "FOREIGN KEY")
	})
}

func TestInsertAndSelectByID(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)
	users := sch.Tables[0]

	id := insertUser(t, database, users, "Alice", "alice@example.com")

	row, err := SelectByID(database, users, id)
	require.NoError(t, err)
	require.NotNil(t, row)

	assert.Equal(t, "Alice", row["name"])
	assert.Equal(t, "alice@example.com", row["email"])
	assert.Equal(t, int64(30), row["age"])
	assert.Equal(t, true, row["active"])
	assert.Equal(t, 9.5, row["score"])
	assert.Equal(t, "2026-01-02T03:04:05Z", row["joined"])
}

func TestInsertErrors(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)
	users := sch.Tables[0]

	t.Run("missing required column", func(t *testing.T) {
		_, err := Insert(database, users, map[string]any{"email": "x@example.com"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "NOT NULL")
	})

	t.Run("unique violation", func(t *testing.T) {
		insertUser(t, database, users, "Alice", "dup@example.com")
		_, err := Insert(database, users, map[string]any{
			"name": "Bob", "email": "dup@example.com",
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "UNIQUE")
	})

	t.Run("foreign key violation", func(t *testing.T) {
		posts := sch.Tables[1]
		_, err := Insert(database, posts, map[string]any{
			"title": "orphan post", "user_id": int64(999),
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "FOREIGN KEY")
	})
}

func TestSelectByIDNotFound(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)

	row, err := SelectByID(database, sch.Tables[0], 999)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestSelectAll(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)
	users := sch.Tables[0]

	t.Run("empty table", func(t *testing.T) {
		rows, total, err := SelectAll(database, users, 1, 20, "", "")
		require.NoError(t, err)
		assert.Empty(t, rows)
		assert.Equal(t, 0, total)
	})

	t.Run("returns all rows sorted by id", func(t *testing.T) {
		insertUser(t, database, users, "Alice", "alice@example.com")
		insertUser(t, database, users, "Bob", "bob@example.com")
		insertUser(t, database, users, "Carol", "carol@example.com")

		rows, total, err := SelectAll(database, users, 1, 20, "", "")
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		require.Len(t, rows, 3)
		assert.Equal(t, "Alice", rows[0]["name"])
		assert.Equal(t, "Bob", rows[1]["name"])
		assert.Equal(t, "Carol", rows[2]["name"])
	})

	t.Run("pagination", func(t *testing.T) {
		page, total, err := SelectAll(database, users, 2, 2, "", "")
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		require.Len(t, page, 1)
		assert.Equal(t, "Carol", page[0]["name"])
	})

	t.Run("sort and order", func(t *testing.T) {
		rows, _, err := SelectAll(database, users, 1, 20, "age", "desc")
		require.NoError(t, err)
		require.Len(t, rows, 3)
		assert.Equal(t, int64(30), rows[0]["age"])
	})

	t.Run("invalid sort falls back to id", func(t *testing.T) {
		rows, _, err := SelectAll(database, users, 1, 20, "nope", "weird")
		require.NoError(t, err)
		require.Len(t, rows, 3)
		assert.Equal(t, "Alice", rows[0]["name"])
	})
}

func TestInsertReturning(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)
	users := sch.Tables[0]

	row, err := InsertReturning(database, users, map[string]any{
		"name": "Alice", "email": "alice@example.com",
		"age": int64(30), "active": true, "score": 9.5,
	})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Alice", row["name"])
	assert.Equal(t, true, row["active"])
	assert.NotNil(t, row["id"])
}

func TestUpdateReturning(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)
	users := sch.Tables[0]
	id := insertUser(t, database, users, "Alice", "alice@example.com")

	t.Run("returns the updated row", func(t *testing.T) {
		row, err := UpdateReturning(database, users, id, map[string]any{"age": int64(31)})
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, int64(31), row["age"])
		assert.Equal(t, "Alice", row["name"])
	})

	t.Run("empty update returns the existing row", func(t *testing.T) {
		row, err := UpdateReturning(database, users, id, map[string]any{})
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, int64(31), row["age"])
	})

	t.Run("missing row returns nil", func(t *testing.T) {
		row, err := UpdateReturning(database, users, 999, map[string]any{"age": int64(40)})
		require.NoError(t, err)
		assert.Nil(t, row)
	})
}

func TestUpdate(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)
	users := sch.Tables[0]

	id := insertUser(t, database, users, "Alice", "alice@example.com")

	t.Run("partial update", func(t *testing.T) {
		require.NoError(t, Update(database, users, id, map[string]any{"age": int64(31)}))

		row, err := SelectByID(database, users, id)
		require.NoError(t, err)
		assert.Equal(t, int64(31), row["age"])
		assert.Equal(t, "Alice", row["name"])
	})

	t.Run("empty update is a no-op", func(t *testing.T) {
		require.NoError(t, Update(database, users, id, map[string]any{}))
	})

	t.Run("missing row is a no-op", func(t *testing.T) {
		require.NoError(t, Update(database, users, 999, map[string]any{"age": int64(40)}))
	})

	t.Run("unique violation", func(t *testing.T) {
		other := insertUser(t, database, users, "Bob", "bob@example.com")
		err := Update(database, users, other, map[string]any{"email": "alice@example.com"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "UNIQUE")
	})
}

func TestDelete(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)
	users := sch.Tables[0]

	t.Run("delete existing", func(t *testing.T) {
		id := insertUser(t, database, users, "Alice", "alice@example.com")
		require.NoError(t, Delete(database, users.Name, id))

		row, err := SelectByID(database, users, id)
		require.NoError(t, err)
		assert.Nil(t, row)
	})

	t.Run("delete missing returns ErrRowNotFound", func(t *testing.T) {
		err := Delete(database, users.Name, 999)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRowNotFound)
	})

	t.Run("delete referenced row fails", func(t *testing.T) {
		posts := sch.Tables[1]
		userID := insertUser(t, database, users, "Carol", "carol@example.com")
		_, err := Insert(database, posts, map[string]any{"title": "post", "user_id": userID})
		require.NoError(t, err)

		err = Delete(database, users.Name, userID)
		require.Error(t, err)
		assert.ErrorContains(t, err, "FOREIGN KEY")
	})
}

func TestTablesEmpty(t *testing.T) {
	database := openTestDB(t)
	sch := createTestTables(t, database)
	users := sch.Tables[0]

	t.Run("fresh tables are empty", func(t *testing.T) {
		empty, err := TablesEmpty(database, sch.Tables)
		require.NoError(t, err)
		assert.True(t, empty)
	})

	t.Run("any populated table is not empty", func(t *testing.T) {
		insertUser(t, database, users, "Alice", "alice@example.com")

		empty, err := TablesEmpty(database, sch.Tables)
		require.NoError(t, err)
		assert.False(t, empty)
	})
}

func TestConvertValue(t *testing.T) {
	sch := createTestTables(t, openTestDB(t))
	users := sch.Tables[0]
	boolCols := BoolColumnSet(users)

	assert.Equal(t, true, ConvertValue("active", int64(1), boolCols))
	assert.Equal(t, false, ConvertValue("active", int64(0), boolCols))
	assert.Equal(t, "text", ConvertValue("name", []byte("text"), boolCols))
	assert.Equal(t, int64(5), ConvertValue("age", int64(5), boolCols))
}

func TestQuoteIdent(t *testing.T) {
	assert.Equal(t, `"users"`, quoteIdent("users"))
	assert.Equal(t, `"a""b"`, quoteIdent(`a"b`))
}

func TestSQLType(t *testing.T) {
	cases := []struct {
		columnType string
		want       string
	}{
		{"string", "TEXT"},
		{"datetime", "TEXT"},
		{"int", "INTEGER"},
		{"bool", "INTEGER"},
		{"float", "REAL"},
		{"unknown", "TEXT"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, sqlType(tc.columnType), "type %s", tc.columnType)
	}
}

func TestSanitizeSort(t *testing.T) {
	sch := createTestTables(t, openTestDB(t))
	users := sch.Tables[0]

	assert.Equal(t, "id", sanitizeSort(users, ""))
	assert.Equal(t, "id", sanitizeSort(users, "nope"))
	assert.Equal(t, "id", sanitizeSort(users, "id"))
	assert.Equal(t, "age", sanitizeSort(users, "age"))
}

func TestSanitizeOrder(t *testing.T) {
	assert.Equal(t, "asc", sanitizeOrder(""))
	assert.Equal(t, "desc", sanitizeOrder("DESC"))
	assert.Equal(t, "asc", sanitizeOrder("asc"))
	assert.Equal(t, "asc", sanitizeOrder("weird"))
}
