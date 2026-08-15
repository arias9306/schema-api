package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/schema"
	"github.com/arias9306/schema-api/validation"
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
	mux.HandleFunc("GET /{table}/{id}", h.Get)
	mux.HandleFunc("POST /{table}", h.Create)
	mux.HandleFunc("PUT /{table}/{id}", h.Update)
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

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("table")
	_, ok := h.findTable(tableName)

	if !ok {
		writeError(w, http.StatusNotFound, "table not found: %s", tableName)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	row, err := db.SelectByID(h.db, tableName, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed: %v", err)
		return
	}

	if row == nil {
		writeError(w, http.StatusNotFound, "row not found")
		return
	}

	writeJSON(w, http.StatusOK, row)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("table")
	table, ok := h.findTable(tableName)

	if !ok {
		writeError(w, http.StatusNotFound, "table not found: %s", tableName)
		return
	}

	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}

	errors, cleaned := validation.ValidateCreate(table, data)
	if errors.HasErrors() {
		writeJSON(w, http.StatusUnprocessableEntity, errors)
		return
	}

	id, err := db.Insert(h.db, table, cleaned)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeError(w, http.StatusConflict, "unique constraint violation")
			return
		}

		if strings.Contains(err.Error(), "FOREIGN KEY constraint") {
			writeError(w, http.StatusBadRequest, "foreign key violation")
			return
		}

		writeError(w, http.StatusInternalServerError, "failed to retrieve create row")
		return
	}

	row, err := db.SelectByID(h.db, tableName, id)
	if err != nil || row == nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve created row")
		return
	}

	writeJSON(w, http.StatusCreated, row)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("table")
	table, ok := h.findTable(tableName)
	if !ok {
		writeError(w, http.StatusNotFound, "table not found: %s", tableName)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
	}

	existing, err := db.SelectByID(h.db, tableName, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed: %v", err)
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "row not found")
		return
	}

	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}

	errors := validation.ValidateUpdate(table, data)
	if errors.HasErrors() {
		writeJSON(w, http.StatusUnprocessableEntity, errors)
		return
	}

	if err := db.Update(h.db, table, id, data); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeError(w, http.StatusConflict, "unique constraint violation")
			return
		}
		if strings.Contains(err.Error(), "FOREIGN KEY constraint") {
			writeError(w, http.StatusBadRequest, "foreign key violation")
			return
		}

		writeError(w, http.StatusInternalServerError, "update failed: %v", err)
		return
	}

	row, err := db.SelectByID(h.db, tableName, id)
	if err != nil || row == nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve update rows")
		return
	}

	writeJSON(w, http.StatusOK, row)
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
