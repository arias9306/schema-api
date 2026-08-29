// Package api provides HTTP handlers for the schema API.
package api

import (
	"cmp"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/httputil"
	"github.com/arias9306/schema-api/schema"
	"github.com/arias9306/schema-api/validation"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const maxBodyBytes = 1 << 20

type Handler struct {
	db     *sql.DB
	schema *schema.Schema
	tables map[string]schema.Table
}

func NewAPIHandler(database *sql.DB, sch *schema.Schema) *Handler {
	tables := make(map[string]schema.Table, len(sch.Tables))
	for _, table := range sch.Tables {
		tables[table.Name] = table
	}
	return &Handler{db: database, schema: sch, tables: tables}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /{table}", h.List)
	mux.HandleFunc("GET /{table}/{id}", h.Get)
	mux.HandleFunc("POST /{table}", h.Create)
	mux.HandleFunc("PUT /{table}/{id}", h.Update)
	mux.HandleFunc("DELETE /{table}/{id}", h.Delete)
}

func (h *Handler) RegisterTableEndpoints(mux *http.ServeMux) error {
	seen := make(map[string]bool, len(h.schema.TableEndpoints))
	for i := range h.schema.TableEndpoints {
		ep := &h.schema.TableEndpoints[i]
		pattern := httputil.RouteKey(ep.Method, ep.Path)
		if seen[pattern] {
			return fmt.Errorf("duplicate table endpoint pattern %q", pattern)
		}
		seen[pattern] = true
	}

	for i := range h.schema.TableEndpoints {
		ep := &h.schema.TableEndpoints[i]
		pattern := httputil.RouteKey(ep.Method, ep.Path)
		mux.HandleFunc(pattern, h.handlerFor(i))
	}

	return nil
}

func (h *Handler) handlerFor(i int) http.HandlerFunc {
	ep := &h.schema.TableEndpoints[i]
	table := h.tables[ep.Tables[0]]

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := httputil.BuildRequestContext(r, schema.ParamNames(ep.Path), false)

		query, params, err := db.BuildQuery(*ep, h.tables, ctx)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "building query: %v", err)
			return
		}

		rows, err := h.db.Query(query, params...)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "query failed: %v", err)
			return
		}
		defer rows.Close()

		results, err := scanRows(rows, table)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "scanning rows: %v", err)
			return
		}

		rendered := db.BuildTableEndpointResponse(ep.Response, results)

		status := cmp.Or(ep.Status, http.StatusOK)

		for name, value := range ep.Headers {
			w.Header().Set(name, value)
		}

		httputil.WriteJSON(w, status, rendered)
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("table")
	table, ok := h.tables[tableName]

	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "table not found: %s", tableName)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	rows, total, err := db.SelectAll(h.db, table, page, limit, sort, order)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "query failed: %v", err)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Page", strconv.Itoa(page))
	w.Header().Set("X-Limit", strconv.Itoa(limit))
	httputil.WriteJSON(w, http.StatusOK, rows)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("table")
	table, ok := h.tables[tableName]

	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "table not found: %s", tableName)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	row, err := db.SelectByID(h.db, table, id)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "query failed: %v", err)
		return
	}

	if row == nil {
		httputil.WriteError(w, http.StatusNotFound, "row not found")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, row)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("table")
	table, ok := h.tables[tableName]

	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "table not found: %s", tableName)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}

	errors, cleaned := validation.ValidateCreate(table, data)
	if errors.HasErrors() {
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, errors)
		return
	}

	id, err := db.Insert(h.db, table, cleaned)
	if err != nil {
		if errCode := constraintCode(err); errCode != nil {
			writeConstraintError(w, *errCode)
			return
		}

		httputil.WriteError(w, http.StatusInternalServerError, "failed to insert row: %v", err)
		return
	}

	row, err := db.SelectByID(h.db, table, id)
	if err != nil || row == nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to retrieve created row")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, row)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("table")
	table, ok := h.tables[tableName]
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "table not found: %s", tableName)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := db.SelectByID(h.db, table, id)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "query failed: %v", err)
		return
	}
	if existing == nil {
		httputil.WriteError(w, http.StatusNotFound, "row not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}

	errors := validation.ValidateUpdate(table, data)
	if errors.HasErrors() {
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, errors)
		return
	}

	if err := db.Update(h.db, table, id, data); err != nil {
		if errCode := constraintCode(err); errCode != nil {
			writeConstraintError(w, *errCode)
			return
		}

		httputil.WriteError(w, http.StatusInternalServerError, "update failed: %v", err)
		return
	}

	row, err := db.SelectByID(h.db, table, id)
	if err != nil || row == nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to retrieve updated row")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, row)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("table")
	if _, ok := h.tables[tableName]; !ok {
		httputil.WriteError(w, http.StatusNotFound, "table not found: %s", tableName)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := db.Delete(h.db, tableName, id); err != nil {
		if errCode := constraintCode(err); errCode != nil {
			if *errCode == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY {
				httputil.WriteError(w, http.StatusConflict, "cannot delete: referenced by other rows")
				return
			}
			writeConstraintError(w, *errCode)
			return
		}

		if errors.Is(err, db.ErrRowNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "row not found")
			return
		}

		httputil.WriteError(w, http.StatusInternalServerError, "delete failed: %v", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func constraintCode(err error) *int {
	sqliteErr, ok := errors.AsType[*sqlite.Error](err)
	if !ok {
		return nil
	}

	code := sqliteErr.Code()
	switch code {
	case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3.SQLITE_CONSTRAINT_NOTNULL, sqlite3.SQLITE_CONSTRAINT_CHECK:
		return &code
	case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
		return &code
	}

	return nil
}

func writeConstraintError(w http.ResponseWriter, code int) {
	switch code {
	case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
		httputil.WriteError(w, http.StatusConflict, "unique constraint violation")
	case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
		httputil.WriteError(w, http.StatusBadRequest, "foreign key violation")
	default:
		httputil.WriteError(w, http.StatusBadRequest, "constraint violation")
	}
}

func scanRows(rows *sql.Rows, table schema.Table) ([]map[string]any, error) {
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
			row[col] = db.ConvertValue(col, values[i], table)
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
