package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/arias9306/schema-api/schema"
)

func main() {

	schemaPath := flag.String("schema", "", "Path to JSON schema file")
	rows := flag.Int("rows", 100, "Number of fake rows per table")
	port := flag.Int("port", 8080, "Server port")

	flag.Parse()

	if *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "error: --schema is required")
		flag.Usage()
		os.Exit(1)
	}

	schema, err := schema.ParseSchema(*schemaPath)
	if err != nil {
		log.Fatalf("failed to parse schema: %v", err)
	}

	fmt.Println(*port)
	fmt.Println(*rows)

	// Init DB

	for _, table := range schema.Tables {
		// Create Table

		fmt.Println(table.Name)
	}

}
