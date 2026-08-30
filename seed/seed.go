// Package seed generates and inserts sample data for schema-defined tables.
package seed

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/fakegen"
	"github.com/arias9306/schema-api/schema"
)

// batchSize bounds how many inserts share a single transaction to avoid holding
// a write lock for too long while still batching the common case.
const maxUniqueAttempts = 1000
const batchSize = 1000

func Seed(database *sql.DB, schema *schema.Schema, rowPerTable int) error {
	topologicalList, err := topologicalSort(schema.Tables)

	if err != nil {
		return fmt.Errorf("dependency order: %w", err)
	}

	parentIDs := map[string][]int64{}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, table := range topologicalList {

		uniqueSets := map[string]map[string]bool{}
		for _, column := range table.Columns {
			if column.Unique {
				uniqueSets[column.Name] = map[string]bool{}
			}
		}

		tx, err := database.Begin()
		if err != nil {
			return fmt.Errorf("beginning transaction for %s: %w", table.Name, err)
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		for i := range rowPerTable {
			row := map[string]any{}

			for _, column := range table.Columns {
				if column.ForeignKey != nil {
					foreingKeys := parentIDs[column.ForeignKey.Table]

					if len(foreingKeys) == 0 {
						continue
					}

					row[column.Name] = foreingKeys[rng.Intn(len(foreingKeys))]
					continue
				}

				value, err := generateValue(rng, column, uniqueSets[column.Name])
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
			id, err := db.Insert(tx, table, row)
			if err != nil {
				return fmt.Errorf("seeding %s row %d: %w", table.Name, i, err)
			}

			parentIDs[table.Name] = append(parentIDs[table.Name], id)

			if (i+1)%batchSize == 0 {
				if err := tx.Commit(); err != nil {
					return fmt.Errorf("committing %s batch: %w", table.Name, err)
				}
				tx, err = database.Begin()
				if err != nil {
					return fmt.Errorf("beginning transaction for %s: %w", table.Name, err)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing %s: %w", table.Name, err)
		}
		committed = true
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
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

func generateValue(rng *rand.Rand, column schema.Column, used map[string]bool) (any, error) {
	for attempt := 0; ; attempt++ {
		value, err := fakegen.Value(rng, columnToSpec(column))
		if err != nil {
			return nil, err
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

func columnToSpec(column schema.Column) fakegen.Spec {
	return fakegen.Spec{
		Type:      column.Type,
		Min:       column.Min,
		Max:       column.Max,
		MinLength: column.MinLength,
		MaxLength: column.MaxLength,
		Regex:     column.Regex,
		Format:    resolveFormat(column),
		Default:   column.Default,
	}
}
