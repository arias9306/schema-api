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
