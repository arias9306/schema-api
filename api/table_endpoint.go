package api

import (
	"cmp"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/schema"
)

type TableEndpointHandler struct {
	database *sql.DB
	schema   *schema.Schema
	tables   map[string]schema.Table
}

func NewTableEndpointHandler(database *sql.DB, sch *schema.Schema) *TableEndpointHandler {
	tables := make(map[string]schema.Table, len(sch.Tables))
	for _, table := range sch.Tables {
		tables[table.Name] = table
	}
	return &TableEndpointHandler{database: database, schema: sch, tables: tables}
}

func (h *TableEndpointHandler) Register(mux *http.ServeMux) error {
	seen := make(map[string]bool, len(h.schema.TableEndpoints))
	for i := range h.schema.TableEndpoints {
		ep := &h.schema.TableEndpoints[i]
		pattern := strings.ToUpper(strings.TrimSpace(ep.Method)) + " " + ep.Path
		if seen[pattern] {
			return fmt.Errorf("duplicate table endpoint pattern %q", pattern)
		}
		seen[pattern] = true
	}

	for i := range h.schema.TableEndpoints {
		ep := &h.schema.TableEndpoints[i]
		pattern := strings.ToUpper(strings.TrimSpace(ep.Method)) + " " + ep.Path
		mux.HandleFunc(pattern, h.handlerFor(i))
	}

	return nil
}

func (h *TableEndpointHandler) handlerFor(i int) http.HandlerFunc {
	ep := &h.schema.TableEndpoints[i]

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := map[string]string{}

		for _, name := range ep.ParamNames() {
			ctx["path."+name] = r.PathValue(name)
		}

		for name, values := range r.URL.Query() {
			if len(values) > 0 {
				ctx["query."+name] = values[0]
			}
		}

		for name, values := range r.Header {
			if len(values) > 0 {
				ctx["header."+strings.ToLower(name)] = values[0]
			}
		}

		query, params, err := db.BuildQuery(*ep, h.tables, ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "building query: %v", err)
			return
		}

		rows, err := h.database.Query(query, params...)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query failed: %v", err)
			return
		}
		defer rows.Close()

		results, err := scanRows(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "scanning rows: %v", err)
			return
		}

		rendered := db.BuildTableEndpointResponse(ep.Response, results)

		status := cmp.Or(ep.Status, http.StatusOK)

		for name, value := range ep.Headers {
			w.Header().Set(name, value)
		}

		writeJSON(w, status, rendered)
	}
}

func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		row := make(map[string]any, len(cols))
		for i, col := range cols {
			v := values[i]
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			row[col] = v
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
