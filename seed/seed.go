// Package seed generates and inserts sample data for schema-defined tables.
package seed

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/arias9306/schema-api/schema"
)

const maxUniqueAttempts = 1000

func Seed(db *sql.DB, schema *schema.Schema, rowPerTable int) error {
	topologicalList, err := topologicalSort(schema.Tables)

	if err != nil {
		return fmt.Errorf("dependency order: %w", err)
	}

	parentIDs := map[string][]int64{}

	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, table := range topologicalList {

		uniqueSets := map[string]map[string]bool{}
		for _, column := range table.Columns {
			if column.Unique {
				uniqueSets[column.Name] = map[string]bool{}
			}
		}

		for i := range rowPerTable {
			row := map[string]any{}

			for _, column := range table.Columns {
				if column.ForeignKey != nil {
					foreingKeys := parentIDs[column.ForeignKey.Table]

					if len(foreingKeys) == 0 {
						continue
					}

					row[column.Name] = foreingKeys[randomizer.Intn(len(foreingKeys))]
					continue
				}

				value, err := generateValue(randomizer, column, uniqueSets[column.Name])
				if err != nil {
					return fmt.Errorf("seeding %s row %d column %s: %w", table.Name, i, column.Name, err)
				}

				row[column.Name] = value
				if column.Unique {
					valueString := fmt.Sprintf("%v", value)
					uniqueSets[column.Name][valueString] = true
				}
			}

			// Insert Row
			id, err := insertRow(db, table.Name, row)
			if err != nil {
				return fmt.Errorf("seeding %s row %d: %w", table.Name, i, err)
			}

			parentIDs[table.Name] = append(parentIDs[table.Name], id)
		}
	}

	return nil
}

// Kahn's algorithm
func topologicalSort(tables []schema.Table) ([]schema.Table, error) {

	nameToTable := map[string]schema.Table{}
	inDegree := map[string]int{}
	dependents := map[string][]string{}

	for _, table := range tables {
		nameToTable[table.Name] = table
		inDegree[table.Name] = 0
	}

	for _, table := range tables {
		for _, column := range table.Columns {
			if column.ForeignKey != nil {
				parent := column.ForeignKey.Table

				if _, ok := nameToTable[parent]; ok {
					dependents[parent] = append(dependents[parent], table.Name)
					inDegree[table.Name]++
				}
			}
		}
	}

	queue := []string{}

	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	result := []schema.Table{}
	visited := map[string]bool{}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		if visited[name] {
			continue
		}

		visited[name] = true
		result = append(result, nameToTable[name])

		for _, dependent := range dependents[name] {
			inDegree[dependent]--

			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(tables) {
		return nil, fmt.Errorf("circula dependency detected")
	}

	return result, nil
}

func generateValue(randomizer *rand.Rand, column schema.Column, used map[string]bool) (any, error) {
	for attempt := 0; ; attempt++ {
		var value any

		switch column.Type {
		case "string":
			value = generateString(randomizer, column)
		}

		if column.Unique {
			valueString := fmt.Sprintf("%v", value)

			if used[valueString] {
				if attempt >= maxUniqueAttempts {
					return nil, fmt.Errorf("could not generate a unique value for %s after %d attempts,", column.Name, maxUniqueAttempts)
				}
				continue
			}
		}

		return value, nil
	}
}

func generateString(randomizer *rand.Rand, column schema.Column) string {

	if format := resolveFormat(column); format != "" {
		if generator, ok := formatGenerators[format]; ok {
			value := generator(randomizer)
			if !violatesExplicitLength(column, value) {
				return value
			}
		}
	}

	minLength := 5
	maxLength := 30

	if column.MinLength != nil {
		minLength = *column.MinLength
	}

	if column.MaxLength != nil {
		maxLength = *column.MaxLength
	}

	if minLength > maxLength {
		minLength, maxLength = maxLength, minLength
	}

	length := minLength
	if maxLength > minLength {
		length += randomizer.Intn(maxLength - minLength + 1)
	}

	return randomString(randomizer, length)
}

func violatesExplicitLength(column schema.Column, value string) bool {
	if column.MinLength != nil && len(value) < *column.MinLength {
		return true
	}
	if column.MaxLength != nil && len(value) > *column.MaxLength {
		return true
	}

	return false
}

func randomString(randomizer *rand.Rand, n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[randomizer.Intn(len(letters))]
	}
	return string(b)
}

func insertRow(db *sql.DB, tableName string, data map[string]any) (int64, error) {
	columns := make([]string, 0, len(data))
	values := make([]any, 0, len(data))

	for key, value := range data {
		columns = append(columns, key)
		values = append(values, value)
	}

	placholders := make([]string, len(columns))

	for i := range placholders {
		placholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, joinColumns(columns), joinColumns(placholders))

	result, err := db.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func joinColumns(columns []string) string {
	var s strings.Builder
	for i, column := range columns {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(column)
	}
	return s.String()
}
