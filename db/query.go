// Package db provides database helpers for schema-backed tables.
package db

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/arias9306/schema-api/schema"
)

const aliasSep = "__"

var refRegex = regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

func aliasOf(table, column string) string {
	return table + aliasSep + column
}

func BuildSelectColumns(response any, tables []string, tableMap map[string]schema.Table) string {
	refs := collectColumnRefs(response)

	columns := make([]string, 0, len(refs)+len(tables))
	for _, table := range tables {
		columns = append(columns, columnAlias(table, "id"))
		for ref := range refs {
			parts := strings.SplitN(ref, ".", 2)
			if len(parts) != 2 {
				continue
			}
			if parts[0] == table && parts[1] != "id" {
				columns = append(columns, columnAlias(table, parts[1]))
			}
		}
	}

	return strings.Join(columns, ", ")
}

func columnAlias(table, column string) string {
	return fmt.Sprintf("%s.%s AS %s", quoteIdent(table), quoteIdent(column), quoteIdent(aliasOf(table, column)))
}

func collectColumnRefs(template any) map[string]bool {
	refs := map[string]bool{}
	walkRefs(template, refs)
	return refs
}

func walkRefs(v any, refs map[string]bool) {
	switch val := v.(type) {
	case string:
		for _, m := range refRegex.FindAllStringSubmatch(val, -1) {
			expr := strings.TrimSpace(m[1])
			if strings.Contains(expr, ".") {
				refs[expr] = true
			}
		}
	case []any:
		for _, item := range val {
			walkRefs(item, refs)
		}
	case map[string]any:
		for _, item := range val {
			walkRefs(item, refs)
		}
	}
}

func InferJoins(tables []string, tableMap map[string]schema.Table) ([]schema.Join, error) {
	joins := make([]schema.Join, 0, len(tables)-1)

	for i := 0; i < len(tables)-1; i++ {
		left := tables[i]
		right := tables[i+1]

		leftTable, ok := tableMap[left]
		if !ok {
			return nil, fmt.Errorf("unknown table %q", left)
		}
		rightTable, ok := tableMap[right]
		if !ok {
			return nil, fmt.Errorf("unknown table %q", right)
		}

		if join, found := findJoin(left, right, rightTable); found {
			joins = append(joins, join)
			continue
		}
		if join, found := findJoin(right, left, leftTable); found {
			joins = append(joins, join)
			continue
		}

		return nil, fmt.Errorf("cannot infer join between %q and %q: no foreign key relationship", left, right)
	}

	return joins, nil
}

func findJoin(parent, child string, childTable schema.Table) (schema.Join, bool) {
	for _, column := range childTable.Columns {
		if column.ForeignKey == nil || column.ForeignKey.Table != parent {
			continue
		}
		refColumn := column.ForeignKey.Column
		if refColumn == "" {
			refColumn = "id"
		}
		return schema.Join{
			Type: "INNER",
			On: schema.JoinCondition{
				Local:   parent + ".id",
				Foreign: child + "." + column.Name,
			},
		}, true
	}
	return schema.Join{}, false
}

// BuildJoinClause converts a slice of joins into the SQL JOIN fragments.
func BuildJoinClause(joins []schema.Join) string {
	var b strings.Builder

	for _, join := range joins {
		joinType := strings.ToUpper(strings.TrimSpace(join.Type))
		if joinType == "" {
			joinType = "INNER"
		}

		local := splitRef(join.On.Local)
		foreign := splitRef(join.On.Foreign)

		b.WriteString(" ")
		b.WriteString(joinType)
		b.WriteString(" JOIN ")
		b.WriteString(quoteIdent(foreign.table))
		b.WriteString(" ON ")
		b.WriteString(quoteIdent(local.table))
		b.WriteString(".")
		b.WriteString(quoteIdent(local.column))
		b.WriteString(" = ")
		b.WriteString(quoteIdent(foreign.table))
		b.WriteString(".")
		b.WriteString(quoteIdent(foreign.column))
	}

	return b.String()
}

type tableColumn struct {
	table  string
	column string
}

func splitRef(ref string) tableColumn {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) != 2 {
		return tableColumn{}
	}
	return tableColumn{table: parts[0], column: parts[1]}
}

func InterpolateWhere(conditions []string, ctx map[string]string) (string, []any, error) {
	parts := make([]string, 0, len(conditions))
	params := make([]any, 0, len(conditions))

	for _, cond := range conditions {
		var b strings.Builder
		last := 0
		matches := refRegex.FindAllStringSubmatchIndex(cond, -1)

		for _, m := range matches {
			start, end := m[0], m[1]
			expr := strings.TrimSpace(cond[m[2]:m[3]])

			b.WriteString(cond[last:start])
			b.WriteString("?")

			value, ok := ctx[expr]
			if !ok {
				value = ""
			}
			params = append(params, value)
			last = end
		}
		b.WriteString(cond[last:])

		if strings.TrimSpace(b.String()) != "" {
			parts = append(parts, b.String())
		}
	}

	return strings.Join(parts, " AND "), params, nil
}

func BuildQuery(ep schema.TableEndpoint, tableMap map[string]schema.Table, ctx map[string]string) (string, []any, error) {
	if len(ep.Tables) == 0 {
		return "", nil, fmt.Errorf("table endpoint has no tables")
	}

	columns := BuildSelectColumns(ep.Response, ep.Tables, tableMap)

	joins := ep.Joins
	if len(joins) == 0 {
		inferred, err := InferJoins(ep.Tables, tableMap)
		if err != nil {
			return "", nil, err
		}
		joins = inferred
	}

	joinClause := BuildJoinClause(joins)

	whereClause, params, err := InterpolateWhere(ep.Where, ctx)
	if err != nil {
		return "", nil, err
	}

	var b strings.Builder
	b.WriteString("SELECT ")
	b.WriteString(columns)
	b.WriteString(" FROM ")
	b.WriteString(quoteIdent(ep.Tables[0]))
	b.WriteString(joinClause)

	if whereClause != "" {
		b.WriteString(" WHERE ")
		b.WriteString(whereClause)
	}

	if ep.OrderBy != "" {
		b.WriteString(" ORDER BY ")
		b.WriteString(ep.OrderBy)
	}

	if ep.Limit != nil {
		b.WriteString(" LIMIT ?")
		params = append(params, *ep.Limit)
	}

	return b.String(), params, nil
}
