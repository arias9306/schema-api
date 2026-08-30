// Package db provides database helpers for schema-backed tables.
package db

import (
	"cmp"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/arias9306/schema-api/schema"
	_ "modernc.org/sqlite"
)

var ErrRowNotFound = errors.New("row not found")

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	if isMemory(path) {
		// :memory: is a single writer; serialize access and tune for speed.
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		if _, err := db.Exec("PRAGMA synchronous = OFF"); err != nil {
			return nil, fmt.Errorf("disabling synchronous writes: %w", err)
		}
		return db, nil
	}

	// File-backed databases support a small pool and WAL for concurrent reads.
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	pragmas := []struct{ statement, label string }{
		{"PRAGMA journal_mode = WAL", "enabling WAL"},
		{"PRAGMA busy_timeout = 5000", "setting busy timeout"},
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p.statement); err != nil {
			return nil, fmt.Errorf("%s: %w", p.label, err)
		}
	}

	return db, nil
}

func isMemory(path string) bool {
	return !strings.HasPrefix(path, "file:") &&
		(path == "" || path == ":memory:")
}

func CreateTable(db *sql.DB, table schema.Table) error {
	cols := []string{"id INTEGER PRIMARY KEY AUTOINCREMENT"}

	for _, column := range table.Columns {
		columnDef := quoteIdent(column.Name) + " " + sqlType(column.Type)

		if column.Required {
			columnDef += " NOT NULL"
		}

		if column.Unique {
			columnDef += " UNIQUE"
		}

		if column.ForeignKey != nil {
			refColumn := cmp.Or(column.ForeignKey.Column, "id")
			columnDef += " REFERENCES " + quoteIdent(column.ForeignKey.Table) + " (" + quoteIdent(refColumn) + ")"
		}

		cols = append(cols, columnDef)
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n %s\n)", quoteIdent(table.Name), strings.Join(cols, ",\n "))

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating table %s: %w", table.Name, err)
	}

	return nil
}

func SelectAll(db *sql.DB, table schema.Table, page int, limit int, sort string, order string) ([]map[string]any, int, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(1) FROM %s", quoteIdent(table.Name))

	var total int
	if err := db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting %s: %w", table.Name, err)
	}

	sort = sanitizeSort(table, sort)
	order = sanitizeOrder(order)

	offset := (page - 1) * limit

	query := fmt.Sprintf("SELECT * FROM %s ORDER BY %s %s LIMIT ? OFFSET ?", quoteIdent(table.Name), quoteIdent(sort), order)

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying %s: %w", table.Name, err)
	}

	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, 0, err
	}

	boolCols := BoolColumnSet(table)
	results := []map[string]any{}

	values := make([]any, len(cols))
	valuePtrs := make([]any, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		row := make(map[string]any)
		for i, col := range cols {
			row[col] = ConvertValue(col, values[i], boolCols)
		}
		results = append(results, row)
	}
	return results, total, nil
}

func SelectByID(db *sql.DB, table schema.Table, id int64) (map[string]any, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", quoteIdent(table.Name))

	rows, err := db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("selecting from %s: %w", table.Name, err)
	}

	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	return scanRow(rows, table)
}

// Inserter is implemented by both *sql.DB and *sql.Tx so Insert can run inside
// a transaction (for batched seeding) or directly on the connection pool.
type Inserter interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func Insert(db Inserter, table schema.Table, data map[string]any) (int64, error) {
	columns := make([]string, 0, len(data))
	values := make([]any, 0, len(data))

	for _, column := range table.Columns {
		if value, ok := data[column.Name]; ok {
			columns = append(columns, quoteIdent(column.Name))
			values = append(values, value)
		}
	}

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoteIdent(table.Name), strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	result, err := db.Exec(query, values...)
	if err != nil {
		return 0, fmt.Errorf("inserting into %s: %w", table.Name, err)
	}

	return result.LastInsertId()
}

// InsertReturning inserts a row and returns it in one round-trip using
// RETURNING, avoiding the follow-up SelectByID in the request hot path.
func InsertReturning(db *sql.DB, table schema.Table, data map[string]any) (map[string]any, error) {
	columns := make([]string, 0, len(data))
	values := make([]any, 0, len(data))

	for _, column := range table.Columns {
		if value, ok := data[column.Name]; ok {
			columns = append(columns, quoteIdent(column.Name))
			values = append(values, value)
		}
	}

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING *", quoteIdent(table.Name), strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	rows, err := db.Query(query, values...)
	if err != nil {
		return nil, fmt.Errorf("inserting into %s: %w", table.Name, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("insert into %s returned no row", table.Name)
	}

	return scanRow(rows, table)
}

func Update(db *sql.DB, table schema.Table, id int64, data map[string]any) error {
	sets := []string{}
	values := []any{}

	for _, column := range table.Columns {
		if value, ok := data[column.Name]; ok {
			sets = append(sets, fmt.Sprintf("%s = ?", quoteIdent(column.Name)))
			values = append(values, value)
		}
	}

	if len(sets) == 0 {
		return nil
	}

	values = append(values, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", quoteIdent(table.Name), strings.Join(sets, ", "))

	_, err := db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("updating %s: %w", table.Name, err)
	}

	return nil
}

// UpdateReturning updates a row and returns it in one round-trip using
// RETURNING, avoiding the follow-up SelectByID in the request hot path.
func UpdateReturning(db *sql.DB, table schema.Table, id int64, data map[string]any) (map[string]any, error) {
	sets := []string{}
	values := []any{}

	for _, column := range table.Columns {
		if value, ok := data[column.Name]; ok {
			sets = append(sets, fmt.Sprintf("%s = ?", quoteIdent(column.Name)))
			values = append(values, value)
		}
	}

	if len(sets) == 0 {
		return SelectByID(db, table, id)
	}

	values = append(values, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ? RETURNING *", quoteIdent(table.Name), strings.Join(sets, ", "))

	rows, err := db.Query(query, values...)
	if err != nil {
		return nil, fmt.Errorf("updating %s: %w", table.Name, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	return scanRow(rows, table)
}

func TablesEmpty(db *sql.DB, tables []schema.Table) (bool, error) {
	for _, table := range tables {
		query := fmt.Sprintf("SELECT COUNT(1) FROM %s", quoteIdent(table.Name))

		var count int
		if err := db.QueryRow(query).Scan(&count); err != nil {
			return false, fmt.Errorf("counting %s: %w", table.Name, err)
		}

		if count > 0 {
			return false, nil
		}
	}

	return true, nil
}

func Delete(db *sql.DB, tableName string, id int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", quoteIdent(tableName))
	result, err := db.Exec(query, id)

	if err != nil {
		return fmt.Errorf("deleting from %s: %w", tableName, err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("deleting %s: %w", tableName, ErrRowNotFound)
	}

	return nil
}

func scanRow(rows *sql.Rows, table schema.Table) (map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))

	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	result := make(map[string]any)
	boolCols := BoolColumnSet(table)
	for i, column := range columns {
		result[column] = ConvertValue(column, values[i], boolCols)
	}

	return result, nil
}

func sanitizeSort(table schema.Table, sort string) string {
	if sort == "id" {
		return "id"
	}

	if slices.ContainsFunc(table.Columns, func(c schema.Column) bool {
		return c.Name == sort
	}) {
		return sort
	}

	return "id"
}

func sanitizeOrder(order string) string {
	if order == "" {
		return "asc"
	}

	order = strings.ToLower(order)
	if order != "asc" && order != "desc" {
		return "asc"
	}

	return order
}

// BoolColumnSet returns the set of column names typed as bool for O(1) lookup
// during cell conversion instead of a linear scan over table.Columns.
func BoolColumnSet(table schema.Table) map[string]struct{} {
	boolCols := make(map[string]struct{}, len(table.Columns))
	for _, column := range table.Columns {
		if column.Type == "bool" {
			boolCols[column.Name] = struct{}{}
		}
	}
	return boolCols
}

func ConvertValue(colName string, value any, boolCols map[string]struct{}) any {
	if b, ok := value.([]byte); ok {
		value = string(b)
	}

	if _, isBool := boolCols[colName]; isBool {
		if n, ok := value.(int64); ok {
			return n != 0
		}
	}

	return value
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func sqlType(columnType string) string {
	switch columnType {
	case "string", "datetime": //TODO: add datetime as TEXT for now.
		return "TEXT"
	case "int":
		return "INTEGER"
	case "float":
		return "REAL"
	case "bool":
		return "INTEGER"
	default:
		return "TEXT"
	}

}
