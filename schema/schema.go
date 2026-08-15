// Package schema defines the data model and parsing logic for database schemas.
package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type ForeignKey struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

type Column struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Required   bool        `json:"required,omitempty"`
	Unique     bool        `json:"unique,omitempty"`
	Min        *float64    `json:"min,omitempty"`
	Max        *float64    `json:"max,omitempty"`
	MinLength  *int        `json:"min_length,omitempty"`
	MaxLength  *int        `json:"max_length,omitempty"`
	Regex      string      `json:"regex,omitempty"`
	Format     string      `json:"format,omitempty"`
	Default    any         `json:"default,omitempty"`
	ForeignKey *ForeignKey `json:"foreign_key,omitempty"`
}

type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

type Endpoint struct {
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Status   int               `json:"status"`
	Headers  map[string]string `json:"headers,omitempty"`
	Response any               `json:"response"`
}

type Schema struct {
	Tables    []Table    `json:"tables,omitempty"`
	Endpoints []Endpoint `json:"endpoints,omitempty"`
}

var allowedMethods = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
}

var wildcardPattern = regexp.MustCompile(`\{([^{}]+)\}`)

func (e Endpoint) ParamNames() []string {
	matches := wildcardPattern.FindAllStringSubmatch(e.Path, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}
	return names
}

func (s *Schema) Validate() error {
	if len(s.Tables) == 0 && len(s.Endpoints) == 0 {
		return fmt.Errorf("schema must define at least one of table or endpoint")
	}

	var errors []string
	seen := map[string]int{}

	for i := range s.Endpoints {
		endpoint := &s.Endpoints[i]
		label := fmt.Sprintf("endpoints[%d]", i)

		if strings.TrimSpace(endpoint.Method) == "" {
			errors = append(errors, fmt.Sprintf("%s: method is required", label))
		} else {
			endpoint.Method = strings.ToUpper(strings.TrimSpace(endpoint.Method))
			if !allowedMethods[endpoint.Method] {
				errors = append(errors, fmt.Sprintf("%s (%s %s): invalid method %q: must be one of GET, POST, PUT, PATCH, DELETE", label, endpoint.Method, endpoint.Path, endpoint.Method))
			}
		}

		if endpoint.Path == "" {
			errors = append(errors, fmt.Sprintf("%s: path is required", label))
		} else if !strings.HasPrefix(endpoint.Path, "/") {
			errors = append(errors, fmt.Sprintf("%s (%s %s): path %q must start with /", label, endpoint.Method, endpoint.Path, endpoint.Path))
		}

		if endpoint.Response == nil {
			errors = append(errors, fmt.Sprintf("%s (%s %s): response is required", label, endpoint.Method, endpoint.Path))
		}

		if endpoint.Status == 0 {
			endpoint.Status = 200
		}

		key := endpoint.Method + " " + endpoint.Path
		if first, ok := seen[key]; ok {
			errors = append(errors, fmt.Sprintf("%s (%s): duplicate pattern, already defined at endpoints[%d]", label, key, first))
		} else {
			seen[key] = i
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("invalid schema: %s", strings.Join(errors, "; "))
	}

	return nil
}

func ParseSchema(path string) (*Schema, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("reading schema file: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("parsing schema JSON: %w", err)
	}

	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("validating schema: %w", err)
	}

	return &schema, nil
}
