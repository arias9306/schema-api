package seed

import (
	"database/sql"
	"fmt"

	"github.com/arias9306/schema-api/schema"
)

func Seed(db *sql.DB, schema *schema.Schema, rowPerTable int) error {
	//  add some dictonary of values
	a, b := topologicalSort(schema.Tables)

	if b != nil {
		return b
	}

	for _, name := range a {
		fmt.Println(name.Name)
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
