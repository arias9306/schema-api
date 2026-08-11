package api

import (
	"database/sql"

	"github.com/arias9306/schema-api/schema"
)

type APIHandler struct {
	db     *sql.DB
	schema *schema.Schema
}

func NewAPIHandler(database *sql.DB, schema *schema.Schema) *APIHandler {
	return &APIHandler{db: database, schema: schema}
}
