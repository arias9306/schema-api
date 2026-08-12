package api

import (
	"database/sql"
	"net/http"

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
	mux.HandleFunc("GET /{table}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /{table}/{id}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /{table}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("PUT /{table}/{id}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("DELETE /{table}/{id}", func(w http.ResponseWriter, r *http.Request) {})
}
