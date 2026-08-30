// Package schema defines the data model and parsing logic for database schemas.
package schema

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/arias9306/schema-api/fakegen"
	"github.com/arias9306/schema-api/httputil"
	"github.com/arias9306/schema-api/template"
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
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Status   int               `json:"status"`
	Headers  map[string]string `json:"headers,omitempty"`
	Tables   []string          `json:"tables"`
	Joins    []Join            `json:"joins,omitempty"`
	Where    []string          `json:"where,omitempty"`
	OrderBy  string            `json:"order_by,omitempty"`
	Limit    *int              `json:"limit,omitempty"`
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

var tableColumnRegex = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)`)

var validColumnTypes = map[string]bool{
	"string":   true,
	"int":      true,
	"float":    true,
	"bool":     true,
	"datetime": true,
}

var allowedJoinTypes = map[string]bool{
	"INNER":       true,
	"LEFT":        true,
	"LEFT OUTER":  true,
	"RIGHT":       true,
	"RIGHT OUTER": true,
	"FULL":        true,
	"FULL OUTER":  true,
	"CROSS":       true,
}

// CRUDRoutes lists the conventional CRUD routes generated for every table,
// with the default status code for each.
var CRUDRoutes = []struct {
	Method string
	Suffix string
	Status int
}{
	{"GET", "", 200},
	{"GET", "/{id}", 200},
	{"POST", "", 201},
	{"PUT", "/{id}", 200},
	{"DELETE", "/{id}", 204},
}

// IsValidColumnType reports whether t is a supported column type.
func IsValidColumnType(t string) bool {
	return validColumnTypes[t]
}

func (e Endpoint) ParamNames() []string {
	return ParamNames(e.Path)
}

func (e TableEndpoint) ParamNames() []string {
	return ParamNames(e.Path)
}

// ParamNames returns the path-parameter names declared in a route path,
// e.g. "/users/{id}" yields ["id"].
func ParamNames(path string) []string {
	matches := wildcardPattern.FindAllStringSubmatch(path, -1)
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
	s.validateTableEndpoints(&errors)

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
	parentColumn := cmp.Or(column.ForeignKey.Column, "id")

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

		endpoint.Status = cmp.Or(endpoint.Status, 200)

		key := httputil.RouteKey(endpoint.Method, endpoint.Path)
		if first, ok := seen[key]; ok {
			*errors = append(*errors, fmt.Sprintf("%s (%s): duplicate pattern, already defined at endpoints[%d]", label, key, first))
		} else {
			seen[key] = i
		}
	}
}

func (s *Schema) validateTableEndpoints(errors *[]string) {
	seen := map[string]int{}
	tableMap := map[string]Table{}
	tableColumns := map[string]map[string]bool{}
	for i := range s.Tables {
		t := &s.Tables[i]
		tableMap[t.Name] = *t
		cols := map[string]bool{"id": true}
		for _, c := range t.Columns {
			cols[c.Name] = true
		}
		tableColumns[t.Name] = cols
	}

	crudPatterns := map[string]bool{}
	for _, t := range s.Tables {
		for _, route := range CRUDRoutes {
			crudPatterns[httputil.RouteKey(route.Method, "/"+t.Name+route.Suffix)] = true
		}
	}

	for i := range s.Endpoints {
		key := httputil.RouteKey(s.Endpoints[i].Method, s.Endpoints[i].Path)
		seen[key] = i
	}

	for i := range s.TableEndpoints {
		ep := &s.TableEndpoints[i]
		label := fmt.Sprintf("table_endpoints[%d]", i)

		if strings.TrimSpace(ep.Method) == "" {
			*errors = append(*errors, fmt.Sprintf("%s: method is required", label))
		} else {
			ep.Method = strings.ToUpper(strings.TrimSpace(ep.Method))
			if ep.Method != "GET" {
				*errors = append(*errors, fmt.Sprintf("%s: invalid method %q: table endpoints only support GET", label, ep.Method))
			}
		}

		if strings.TrimSpace(ep.Path) == "" {
			*errors = append(*errors, fmt.Sprintf("%s: path is required", label))
		} else if !strings.HasPrefix(ep.Path, "/") {
			*errors = append(*errors, fmt.Sprintf("%s (%s %s): path %q must start with /", label, ep.Method, ep.Path, ep.Path))
		}

		if ep.Response == nil {
			*errors = append(*errors, fmt.Sprintf("%s (%s %s): response is required", label, ep.Method, ep.Path))
		}

		ep.Status = cmp.Or(ep.Status, 200)

		tablesSet := map[string]bool{}
		for _, t := range ep.Tables {
			if t == "" {
				*errors = append(*errors, fmt.Sprintf("%s (%s %s): table name is required", label, ep.Method, ep.Path))
				continue
			}
			if _, ok := tableMap[t]; !ok {
				*errors = append(*errors, fmt.Sprintf("%s (%s %s): references unknown table %q", label, ep.Method, ep.Path, t))
			}
			if tablesSet[t] {
				*errors = append(*errors, fmt.Sprintf("%s (%s %s): duplicate table %q", label, ep.Method, ep.Path, t))
			}
			tablesSet[t] = true
		}
		if len(ep.Tables) == 0 {
			*errors = append(*errors, fmt.Sprintf("%s (%s %s): at least one table is required", label, ep.Method, ep.Path))
		}

		for _, join := range ep.Joins {
			joinType := strings.ToUpper(strings.TrimSpace(join.Type))
			if joinType == "" {
				joinType = "INNER"
			}
			if !allowedJoinTypes[joinType] {
				*errors = append(*errors, fmt.Sprintf("%s (%s %s): invalid join type %q: must be one of INNER, LEFT, RIGHT, FULL, CROSS (with optional OUTER)", label, ep.Method, ep.Path, join.Type))
			}
			checkJoinRef := func(ref string, which string) {
				left, right := template.SplitRef(ref)
				if !validIdentifier(left) || !validIdentifier(right) {
					*errors = append(*errors, fmt.Sprintf("%s (%s %s): join %s %q must be in table.column format", label, ep.Method, ep.Path, which, ref))
					return
				}
				if !tablesSet[left] {
					*errors = append(*errors, fmt.Sprintf("%s (%s %s): join %s references table %q not listed in tables", label, ep.Method, ep.Path, which, left))
					return
				}
				if !tableColumns[left][right] {
					*errors = append(*errors, fmt.Sprintf("%s (%s %s): join %s references unknown column %q in table %q", label, ep.Method, ep.Path, which, right, left))
				}
			}
			checkJoinRef(join.On.Local, "local")
			checkJoinRef(join.On.Foreign, "foreign")
		}

		for _, cond := range ep.Where {
			if strings.Contains(cond, ";") {
				*errors = append(*errors, fmt.Sprintf("%s (%s %s): where condition must not contain ';'", label, ep.Method, ep.Path))
			}
			if strings.ContainsAny(cond, "()") {
				*errors = append(*errors, fmt.Sprintf("%s (%s %s): where condition must not contain parentheses (subqueries are not allowed)", label, ep.Method, ep.Path))
			}
			stripped := template.RefRegex.ReplaceAllString(cond, "")
			for _, m := range tableColumnRegex.FindAllStringSubmatch(stripped, -1) {
				if !tablesSet[m[1]] {
					*errors = append(*errors, fmt.Sprintf("%s (%s %s): where condition references unknown table %q", label, ep.Method, ep.Path, m[1]))
				}
			}
		}

		if ep.OrderBy != "" {
			s.validateTableEndpointOrderBy(ep, label, tablesSet, tableColumns, errors)
		}

		if ep.Response != nil {
			for _, ref := range template.CollectRefs(ep.Response) {
				left, right := template.SplitRef(ref)
				if !tablesSet[left] {
					continue
				}
				if !tableColumns[left][right] {
					*errors = append(*errors, fmt.Sprintf("%s (%s %s): response references unknown column %q in table %q", label, ep.Method, ep.Path, right, left))
				}
			}
		}

		key := httputil.RouteKey(ep.Method, ep.Path)
		if first, ok := seen[key]; ok {
			*errors = append(*errors, fmt.Sprintf("%s (%s): duplicate pattern, already defined at index %d", label, key, first))
		} else if crudPatterns[key] {
			*errors = append(*errors, fmt.Sprintf("%s (%s): duplicate pattern, shadows a CRUD route", label, key))
		} else {
			seen[key] = i
		}
	}
}

func (s *Schema) validateTableEndpointOrderBy(ep *TableEndpoint, label string, tablesSet map[string]bool, tableColumns map[string]map[string]bool, errors *[]string) {
	parts := strings.Fields(strings.TrimSpace(ep.OrderBy))
	if len(parts) == 0 || len(parts) > 2 {
		*errors = append(*errors, fmt.Sprintf("%s (%s %s): order_by %q must be in table.column[ asc|desc] format", label, ep.Method, ep.Path, ep.OrderBy))
		return
	}

	if len(parts) == 2 {
		dir := strings.ToLower(parts[1])
		if dir != "asc" && dir != "desc" {
			*errors = append(*errors, fmt.Sprintf("%s (%s %s): order_by direction %q must be asc or desc", label, ep.Method, ep.Path, parts[1]))
		}
	}

	left, right := template.SplitRef(parts[0])
	if !validIdentifier(left) || !validIdentifier(right) {
		*errors = append(*errors, fmt.Sprintf("%s (%s %s): order_by %q must reference a table.column", label, ep.Method, ep.Path, parts[0]))
		return
	}
	if !tablesSet[left] {
		*errors = append(*errors, fmt.Sprintf("%s (%s %s): order_by references table %q not listed in tables", label, ep.Method, ep.Path, left))
		return
	}
	if !tableColumns[left][right] {
		*errors = append(*errors, fmt.Sprintf("%s (%s %s): order_by references unknown column %q in table %q", label, ep.Method, ep.Path, right, left))
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
	return slices.ContainsFunc(table.Columns, func(c Column) bool {
		return c.Name == name
	})
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
