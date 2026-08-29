// Package template provides shared helpers for working with the {{ expr }}
// reference syntax used across schema templates, SQL conditions, and mock
// responses.
package template

import (
	"maps"
	"regexp"
	"slices"
	"strings"
)

// RefRegex matches {{ expr }} placeholders, capturing the trimmed expression.
var RefRegex = regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

func CollectRefs(v any) []string {
	refs := map[string]bool{}

	var walk func(v any)
	walk = func(v any) {
		switch val := v.(type) {
		case string:
			for _, m := range RefRegex.FindAllStringSubmatch(val, -1) {
				expr := strings.TrimSpace(m[1])
				if strings.Contains(expr, ".") {
					refs[expr] = true
				}
			}
		case []any:
			for _, item := range val {
				walk(item)
			}
		case map[string]any:
			for _, item := range val {
				walk(item)
			}
		}
	}

	walk(v)

	keys := slices.Collect(maps.Keys(refs))
	slices.Sort(keys)
	return keys
}

func SplitRef(ref string) (table, column string) {
	table, column, _ = strings.Cut(ref, ".")
	return table, column
}
