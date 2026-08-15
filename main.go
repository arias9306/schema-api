package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/arias9306/schema-api/api"
	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/schema"
	"github.com/arias9306/schema-api/seed"
	"github.com/arias9306/schema-api/version"
)

func main() {
	schemaPath := flag.String("schema", "", "Path to JSON schema file")
	rows := flag.Int("rows", 10, "Number of fake rows per table")
	port := flag.Int("port", 8080, "Server port")
	showVersion := flag.Bool("version", false, "Print version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	if *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "error: --schema is required")
		flag.Usage()
		os.Exit(1)
	}

	schema, err := schema.ParseSchema(*schemaPath)
	if err != nil {
		log.Fatalf("failed to parse schema: %v", err)
	}

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

	handler := api.NewAPIHandler(database, schema)
	mux := http.NewServeMux()
	handler.Register(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\nShutting down...")
		server.Close()
	}()

	fmt.Printf("Server running on http://localhost:%d\n", *port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

}
