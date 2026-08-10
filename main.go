package main

import (
	"flag"
	"fmt"
	"os"
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

	fmt.Println(*rows)
	fmt.Println(*port)
	fmt.Println(*schemaPath)
}
