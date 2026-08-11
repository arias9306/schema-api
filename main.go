package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/schema"
	"github.com/arias9306/schema-api/seed"
)

func main() {

	schemaPath := flag.String("schema", "schema.json", "Path to JSON schema file")
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

	fmt.Println(*schema)
	fmt.Println(*port)
	fmt.Println(*rows)

	// TODO: maybe support to add db persistence
	database, err := db.InitDB(":memory:")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	for _, table := range schema.Tables {
		if err := db.CreateTable(database, table); err != nil {
			log.Fatalf("failed to create table %s: %v", table.Name, err)
		}
		fmt.Printf("table %s created.\n", table.Name)
	}

	if *rows > 0 {
		fmt.Printf("seeding %d rows per table...\n", *rows)
		if err := seed.Seed(database, schema, *rows); err != nil {
			log.Fatalf("failed to seed data: %v", err)
		}
		fmt.Println("seeding complete.")
	}

	// api := api.NewAPIHandler(database, schema)

	// mux := http.NewServeMux()

}
