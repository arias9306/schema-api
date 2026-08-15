package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/schema"
)

type Handler struct {
	db     *sql.DB
	schema *schema.Schema
}

func NewAPIHandler(database *sql.DB, schema *schema.Schema) *Handler {
	return &Handler{db: database, schema: schema}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /{table}", h.List)
	mux.HandleFunc("GET /{table}/{id}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /{table}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("PUT /{table}/{id}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("DELETE /{table}/{id}", func(w http.ResponseWriter, r *http.Request) {})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("table")
	table, ok := h.findTable(tableName)

	if !ok {
		writeError(w, http.StatusNotFound, "table not found: %s", tableName)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20 // increase the limit?
	}

	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	rows, total, err := db.SelectAll(h.db, table.Name, page, limit, sort, order)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failied: %v", err)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Page", strconv.Itoa(page))
	w.Header().Set("X-Limit", strconv.Itoa(limit))
	writeJSON(w, http.StatusOK, rows)
}

func (h *Handler) findTable(name string) (schema.Table, bool) {
	for _, table := range h.schema.Tables {
		if table.Name == name {
			return table, true
		}
	}
	return schema.Table{}, false
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
