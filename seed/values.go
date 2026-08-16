package seed

import (
	"strings"

	"github.com/arias9306/schema-api/fakegen"
	"github.com/arias9306/schema-api/schema"
)

func resolveFormat(column schema.Column) string {

	if column.Format != "" {
		return strings.ToLower(column.Format)
	}

	if column.Type != "string" {
		return ""
	}

	return fakegen.ResolveFormat(column.Name)
}
