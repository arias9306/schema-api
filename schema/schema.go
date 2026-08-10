package schema

import (
	"encoding/json"
	"fmt"
	"os"
)

type Column struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Required  bool     `json:"required,omitempty"`
	Unique    bool     `json:"unique,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	Regex     string   `json:"regex,omitempty"`
	Default   any      `json:"default,omitempty"`
	// TODO: add support to foreignkeys...
}

type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

type Schema struct {
	Tables []Table `json:"tables"`
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

	return &schema, nil
}
