// Package validation provides validation functionality for schema tables and columns.
package validation

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/arias9306/schema-api/schema"
)

type ValidationError struct {
	Errors []string `json:"errors"`
}

func (v *ValidationError) Error() string {
	return "validation failed: " + strings.Join(v.Errors, "; ")
}

func (v *ValidationError) Add(format string, args ...any) {
	v.Errors = append(v.Errors, fmt.Sprintf(format, args...))
}

func (v *ValidationError) HasErrors() bool {
	return len(v.Errors) > 0
}

func ValidateCreate(table schema.Table, data map[string]any) (*ValidationError, map[string]any) {
	cleaned := make(map[string]any)
	for key, value := range data {
		cleaned[key] = value
	}

	errors := &ValidationError{}

	for _, column := range table.Columns {
		if _, ok := cleaned[column.Name]; !ok && column.Default != nil {
			cleaned[column.Name] = processDefault(column)
		}
	}

	for _, column := range table.Columns {
		value, ok := cleaned[column.Name]
		if !ok {
			if column.Required {
				errors.Add("%s is required", column.Name)
			}
			continue
		}

		cleaned[column.Name] = value
		if err := validateColumn(column, value); err != "" {
			errors.Add("%s: %s", column.Name, err)
		}
	}

	return errors, cleaned
}

func ValidateUpdate(table schema.Table, data map[string]any) *ValidationError {
	errors := &ValidationError{}

	for _, column := range table.Columns {
		value, ok := data[column.Name]
		if !ok {
			continue
		}
		if err := validateColumn(column, value); err != "" {
			errors.Add("%s: %s", column.Name, err)
		}
	}

	return errors
}

func processDefault(column schema.Column) any {
	if column.Type == "datetime" {
		if s, ok := column.Default.(string); ok && s == "now" {
			return time.Now().UTC().Format(time.RFC3339)
		}
	}

	return column.Default
}

func validateColumn(column schema.Column, value any) string {
	switch column.Type {
	case "string":
		s, ok := value.(string)
		if !ok {
			return "must be a string"
		}

		if column.MinLength != nil && len(s) < *column.MinLength {
			return fmt.Sprintf("length must be at least %d", *column.MinLength)
		}

		if column.MaxLength != nil && len(s) > *column.MaxLength {
			return fmt.Sprintf("length must be at most %d", *column.MaxLength)
		}

		if column.Regex != "" {
			pattern := column.RegexCompiled
			if pattern == nil {
				compiled, err := regexp.Compile(column.Regex)
				if err != nil {
					return fmt.Sprintf("regex error: %v", err)
				}
				pattern = compiled
			}

			if !pattern.MatchString(s) {
				return "does not match pattern"
			}
		}
	case "int":
		f, ok := value.(float64)
		if !ok {
			return "must be a number"
		}

		if f != math.Trunc(f) {
			return "must be an integer"
		}

		v := int(f)
		if column.Min != nil && float64(v) < *column.Min {
			return fmt.Sprintf("must be at least %v", *column.Min)
		}

		if column.Max != nil && float64(v) > *column.Max {
			return fmt.Sprintf("must be at most %v", *column.Max)
		}

	case "float":
		f, ok := value.(float64)
		if !ok {
			return "must be a number"
		}

		if column.Min != nil && f < *column.Min {
			return fmt.Sprintf("must be at least %v", *column.Min)
		}

		if column.Max != nil && f > *column.Max {
			return fmt.Sprintf("must be at most %v", *column.Max)
		}
	case "bool":
		_, ok := value.(bool)
		if !ok {
			return "must be a boolean"
		}

	case "datetime":
		s, ok := value.(string)
		if !ok {
			return "must be a string"
		}

		formats := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
		valid := false

		for _, format := range formats {
			if _, err := time.Parse(format, s); err == nil {
				valid = true
				break
			}
		}

		if !valid {
			return "must be a valid datetime (RFC3339 or YYYY-MM-DD HH:MM:SS)"
		}
	default:
		return fmt.Sprintf("unsupported type: %s", column.Type)
	}

	return ""
}
