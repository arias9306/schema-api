// Package db provides database helpers for schema-backed tables.
package db

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/arias9306/schema-api/fakegen"
)

var responseRefRegex = regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

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
	matches := responseRefRegex.FindAllStringSubmatchIndex(s, -1)

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
	return responseRefRegex.MatchString(trimmed) &&
		responseRefRegex.ReplaceAllString(trimmed, "") == ""
}

var specRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func renderSpec(spec map[string]any) any {
	s := fakegen.Spec{}
	if t, ok := spec["type"].(string); ok {
		s.Type = t
	}
	if f := floatPtr(spec["min"]); f != nil {
		s.Min = f
	}
	if f := floatPtr(spec["max"]); f != nil {
		s.Max = f
	}
	if n := intPtr(spec["min_length"]); n != nil {
		s.MinLength = n
	}
	if n := intPtr(spec["max_length"]); n != nil {
		s.MaxLength = n
	}
	if r, ok := spec["regex"].(string); ok {
		s.Regex = r
	}
	if f, ok := spec["format"].(string); ok {
		s.Format = f
	}
	if d, ok := spec["default"]; ok {
		s.Default = d
	}

	val, err := fakegen.Value(specRand, s)
	if err != nil {
		return nil
	}
	return val
}

func floatPtr(v any) *float64 {
	switch n := v.(type) {
	case float64:
		return &n
	case int:
		f := float64(n)
		return &f
	case int64:
		f := float64(n)
		return &f
	}
	return nil
}

func intPtr(v any) *int {
	switch n := v.(type) {
	case float64:
		i := int(n)
		return &i
	case int:
		return &n
	case int64:
		i := int(n)
		return &i
	}
	return nil
}
