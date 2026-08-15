package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/arias9306/schema-api/schema"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	return db, nil
}

func CreateTable(db *sql.DB, table schema.Table) error {
	cols := []string{"id INTEGER PRIMARY KEY AUTOINCREMENT"}

	for _, column := range table.Columns {
		columnDef := fmt.Sprintf("%s %s", column.Name, sqlType(column.Type))

		if column.Required {
			columnDef += " NOT NULL"
		}

		if column.Unique {
			columnDef += " UNIQUE"
		}

		cols = append(cols, columnDef)
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n %s\n)", table.Name, strings.Join(cols, ",\n "))

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating table %s: %w", table.Name, err)
	}

	return nil
}

func SelectAll(db *sql.DB, tableName string, page int, limit int, sort string, order string) ([]map[string]any, int, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(1) FROM %s", tableName)

	var total int
	if err := db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting %s: %w", tableName, err)
	}

	if sort == "" {
		sort = "id"
	}

	if order == "" {
		order = "asc"
	}
	order = strings.ToLower(order)
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	offset := (page - 1) * limit

	query := fmt.Sprintf("SELECT * FROM %s ORDER BY %s %s LIMIT ? OFFSET ?", tableName, sort, order)

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying %s: %w", tableName, err)
	}

	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, 0, err
	}

	results := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		row := make(map[string]any)
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	return results, total, nil
}

func SelectByID(db *sql.DB, tableName string, id int64) (map[string]any, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tableName)

	rows, err := db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("selecting from %s: %w", tableName, err)
	}

	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	return scanRow(rows)
}

func scanRow(rows *sql.Rows) (map[string]any, error) {
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
	for i, column := range columns {
		value := values[i]

		if b, ok := value.([]byte); ok {
			result[column] = string(b)
		} else {
			result[column] = value
		}
	}

	return result, nil
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
