// Package db provides database helpers for schema-backed tables.
package db

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/arias9306/schema-api/fakegen"
	"github.com/arias9306/schema-api/template"
)

func BuildTableEndpointResponse(template any, rows []map[string]any) any {
	var firstRow map[string]any
	if len(rows) > 0 {
		firstRow = rows[0]
	}
	return renderTemplate(template, rows, firstRow)
}

func renderTemplate(template any, rows []map[string]any, currentRow map[string]any) any {
	switch v := template.(type) {
	case nil, bool, float64, int, int64, string:
		if s, ok := template.(string); ok {
			return resolveString(s, currentRow)
		}
		return template

	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = renderTemplate(item, rows, currentRow)
		}
		return out

	case map[string]any:
		if t, ok := v["type"].(string); ok && t == "array" {
			return renderArray(v, rows)
		}
		if _, ok := v["type"].(string); ok {
			return renderSpec(v)
		}
		out := make(map[string]any, len(v))
		for key, val := range v {
			out[key] = renderTemplate(val, rows, currentRow)
		}
		return out

	default:
		return template
	}
}

func renderArray(spec map[string]any, rows []map[string]any) any {
	items := spec["items"]
	if len(rows) == 0 {
		return []any{}
	}

	out := make([]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, renderTemplate(items, rows, row))
	}
	return out
}

func resolveString(s string, row map[string]any) any {
	if !strings.Contains(s, "{{") {
		return s
	}

	var b strings.Builder
	last := 0
	resolvedAny := false
	matches := template.RefRegex.FindAllStringSubmatchIndex(s, -1)

	for _, m := range matches {
		start, end := m[0], m[1]
		expr := strings.TrimSpace(s[m[2]:m[3]])

		key := strings.Replace(expr, ".", aliasSep, 1)
		b.WriteString(s[last:start])

		if row != nil {
			if val, ok := row[key]; ok {
				b.WriteString(fmt.Sprintf("%v", val))
				resolvedAny = true
				last = end
				continue
			}
		}

		b.WriteString(s[start:end])
		last = end
	}
	b.WriteString(s[last:])

	result := b.String()

	if row == nil && isSinglePlaceholder(s) {
		return nil
	}

	if !resolvedAny {
		return s
	}

	return result
}

func isSinglePlaceholder(s string) bool {
	trimmed := strings.TrimSpace(s)
	return template.RefRegex.MatchString(trimmed) &&
		template.RefRegex.ReplaceAllString(trimmed, "") == ""
}

var specRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func renderSpec(spec map[string]any) any {
	s := fakegen.SpecFromMap(spec, "")

	val, err := fakegen.Value(specRand, s)
	if err != nil {
		return nil
	}
	return val
}
