// Package schema defines the data model and parsing logic for database schemas.
package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/arias9306/schema-api/fakegen"
)

type ForeignKey struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

type Column struct {
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	Required      bool           `json:"required,omitempty"`
	Unique        bool           `json:"unique,omitempty"`
	Min           *float64       `json:"min,omitempty"`
	Max           *float64       `json:"max,omitempty"`
	MinLength     *int           `json:"min_length,omitempty"`
	MaxLength     *int           `json:"max_length,omitempty"`
	Regex         string         `json:"regex,omitempty"`
	RegexCompiled *regexp.Regexp `json:"-"`
	Format        string         `json:"format,omitempty"`
	Default       any            `json:"default,omitempty"`
	ForeignKey    *ForeignKey    `json:"foreign_key,omitempty"`
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

type JoinCondition struct {
	Local   string `json:"local"`
	Foreign string `json:"foreign"`
}

type Join struct {
	Type string        `json:"type"`
	On   JoinCondition `json:"on"`
}

type TableEndpoint struct {
	Path     string            `json:"path"`
	Headers  map[string]string `json:"headers,omitempty"`
	Tables   []string          `json:"tables"`
	Joins    []Join            `json:"joins,omitempty"`
	Where    []string          `json:"where,omitempty"`
	OrderBy  string            `json:"order_by,omitempty"`
	Response any               `json:"response"`
}

type Schema struct {
	Tables         []Table         `json:"tables,omitempty"`
	Endpoints      []Endpoint      `json:"endpoints,omitempty"`
	TableEndpoints []TableEndpoint `json:"table_endpoints,omitempty"`
}

var allowedMethods = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
}

var wildcardPattern = regexp.MustCompile(`\{([^{}]+)\}`)

var identifierPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

var validColumnTypes = map[string]bool{
	"string":   true,
	"int":      true,
	"float":    true,
	"bool":     true,
	"datetime": true,
}

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
	s.validateTables(&errors)
	s.validateEndpoints(&errors)

	if len(errors) > 0 {
		return fmt.Errorf("invalid schema: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (s *Schema) validateTables(errors *[]string) {
	seen := map[string]int{}

	for i := range s.Tables {
		table := &s.Tables[i]
		label := fmt.Sprintf("tables[%d]", i)

		if strings.TrimSpace(table.Name) == "" {
			*errors = append(*errors, fmt.Sprintf("%s: table name is required", label))
		} else if !validIdentifier(table.Name) {
			*errors = append(*errors, fmt.Sprintf("%s: invalid table name %q: must match [A-Za-z][A-Za-z0-9_]*", label, table.Name))
		} else if first, ok := seen[table.Name]; ok {
			*errors = append(*errors, fmt.Sprintf("%s: duplicate table name %q, already defined at tables[%d]", label, table.Name, first))
		} else {
			seen[table.Name] = i
		}

		s.validateColumns(table, label, errors)
	}
}

func (s *Schema) validateColumns(table *Table, tableLabel string, errors *[]string) {
	seen := map[string]int{}

	for j := range table.Columns {
		column := &table.Columns[j]
		label := fmt.Sprintf("%s: column %q", tableLabel, column.Name)

		if strings.TrimSpace(column.Name) == "" {
			*errors = append(*errors, fmt.Sprintf("%s: column name is required", label))
		} else if column.Name == "id" {
			*errors = append(*errors, fmt.Sprintf("%s: column name %q is reserved", label, column.Name))
		} else if !validIdentifier(column.Name) {
			*errors = append(*errors, fmt.Sprintf("%s: invalid column name %q: must match [A-Za-z][A-Za-z0-9_]*", label, column.Name))
		} else if first, ok := seen[column.Name]; ok {
			*errors = append(*errors, fmt.Sprintf("%s: duplicate column, already defined at column %d", label, first))
		} else {
			seen[column.Name] = j
		}

		if !validColumnTypes[column.Type] {
			*errors = append(*errors, fmt.Sprintf("%s: invalid type %q: must be one of string, int, float, bool, datetime", label, column.Type))
		}

		if column.Min != nil && column.Max != nil && *column.Min > *column.Max {
			*errors = append(*errors, fmt.Sprintf("%s: min (%v) must be <= max (%v)", label, *column.Min, *column.Max))
		}

		if column.MinLength != nil && *column.MinLength < 0 {
			*errors = append(*errors, fmt.Sprintf("%s: min_length must be >= 0", label))
		}

		if column.MaxLength != nil && *column.MaxLength < 0 {
			*errors = append(*errors, fmt.Sprintf("%s: max_length must be >= 0", label))
		}

		if column.MinLength != nil && column.MaxLength != nil && *column.MinLength > *column.MaxLength {
			*errors = append(*errors, fmt.Sprintf("%s: min_length (%d) must be <= max_length (%d)", label, *column.MinLength, *column.MaxLength))
		}

		if column.Format != "" && !validFormat(column.Format) {
			*errors = append(*errors, fmt.Sprintf("%s: unknown format %q", label, column.Format))
		}

		if column.Regex != "" {
			compiled, err := regexp.Compile(column.Regex)
			if err != nil {
				*errors = append(*errors, fmt.Sprintf("%s: invalid regex: %v", label, err))
			} else {
				column.RegexCompiled = compiled
			}
		}

		if column.ForeignKey != nil {
			s.validateForeignKey(column, label, errors)
		}
	}
}

func (s *Schema) validateForeignKey(column *Column, label string, errors *[]string) {
	parent := column.ForeignKey.Table
	parentColumn := column.ForeignKey.Column
	if parentColumn == "" {
		parentColumn = "id"
	}

	var parentTable *Table
	for i := range s.Tables {
		if s.Tables[i].Name == parent {
			parentTable = &s.Tables[i]
			break
		}
	}

	if parentTable == nil {
		*errors = append(*errors, fmt.Sprintf("%s: foreign_key references unknown table %q", label, parent))
		return
	}

	if parentColumn != "id" && !columnInTable(parentTable, parentColumn) {
		*errors = append(*errors, fmt.Sprintf("%s: foreign_key references unknown column %q in table %q", label, parentColumn, parent))
	}
}

func (s *Schema) validateEndpoints(errors *[]string) {
	seen := map[string]int{}

	for i := range s.Endpoints {
		endpoint := &s.Endpoints[i]
		label := fmt.Sprintf("endpoints[%d]", i)

		if strings.TrimSpace(endpoint.Method) == "" {
			*errors = append(*errors, fmt.Sprintf("%s: method is required", label))
		} else {
			endpoint.Method = strings.ToUpper(strings.TrimSpace(endpoint.Method))
			if !allowedMethods[endpoint.Method] {
				*errors = append(*errors, fmt.Sprintf("%s (%s %s): invalid method %q: must be one of GET, POST, PUT, PATCH, DELETE", label, endpoint.Method, endpoint.Path, endpoint.Method))
			}
		}

		if endpoint.Path == "" {
			*errors = append(*errors, fmt.Sprintf("%s: path is required", label))
		} else if !strings.HasPrefix(endpoint.Path, "/") {
			*errors = append(*errors, fmt.Sprintf("%s (%s %s): path %q must start with /", label, endpoint.Method, endpoint.Path, endpoint.Path))
		}

		if endpoint.Response == nil {
			*errors = append(*errors, fmt.Sprintf("%s (%s %s): response is required", label, endpoint.Method, endpoint.Path))
		}

		if endpoint.Status == 0 {
			endpoint.Status = 200
		}

		key := endpoint.Method + " " + endpoint.Path
		if first, ok := seen[key]; ok {
			*errors = append(*errors, fmt.Sprintf("%s (%s): duplicate pattern, already defined at endpoints[%d]", label, key, first))
		} else {
			seen[key] = i
		}
	}
}

func validIdentifier(name string) bool {
	return identifierPattern.MatchString(name)
}

func validFormat(format string) bool {
	_, ok := fakegen.FormatGenerators[strings.ToLower(format)]
	return ok
}

func columnInTable(table *Table, name string) bool {
	for _, column := range table.Columns {
		if column.Name == name {
			return true
		}
	}
	return false
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
